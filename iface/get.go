package iface

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

// fileInfo represents syntactic and type information for a file containing
// interfaces to be mocked.
type fileInfo struct {
	pkg             *packages.Package
	sourceFileNodes map[*ast.File]struct{}
	objects         []types.Object
}

// GetAllInterfaces searches the given packages for interfaces annotated with a
// "go:mock" directive, returning text-template-friendly representations grouped
// by output file. If nonempty, defaultOutputFile will be used as the output
// file for interfaces without an explicit output file.
func GetAllInterfaces(pkgs []*packages.Package, defaultOutputFile string) (map[string]File, error) {
	fileInfoByPath := map[string]*fileInfo{}
	for _, pkg := range pkgs {
		for _, fileNode := range pkg.Syntax {
			ast.Inspect(fileNode, func(node ast.Node) bool {
				// A nil node indicates we've finished traversing the AST.
				if node == nil {
					return false
				}
				// Only consider declarations with godoc comments.
				decl, isGenDecl := node.(*ast.GenDecl)
				if !isGenDecl || decl.Doc == nil {
					return true
				}
				// Look for the first go:mock directive in the comments.
				for _, comment := range decl.Doc.List {
					args := strings.TrimPrefix(comment.Text, "//go:mock")
					if args == comment.Text {
						continue
					}

					// Build a qualified output pathname based on the output
					// filename (or a default) and the input filepath.
					var (
						inputPath  = pkg.Fset.File(fileNode.Pos()).Name()
						outputFile = strings.TrimSpace(args)
						outputPath string
					)
					switch {
					case outputFile != "":
						outputPath = filepath.Join(filepath.Dir(inputPath), outputFile)
					case defaultOutputFile != "":
						outputPath = filepath.Join(filepath.Dir(inputPath), defaultOutputFile)
					default:
						outputPath = strings.TrimSuffix(inputPath, ".go") + "_mock.go"
					}

					for _, spec := range decl.Specs {
						// Only consider type declarations.
						spec, isType := spec.(*ast.TypeSpec)
						if !isType {
							continue
						}
						// Get the type info for this declaration.
						object := pkg.TypesInfo.Defs[spec.Name]
						if object == nil {
							continue
						}
						// Create the file info if it doesn't already exist, and
						// add the newly discovered interface.
						if _, fileInfoExists := fileInfoByPath[outputPath]; !fileInfoExists {
							fileInfoByPath[outputPath] = &fileInfo{
								pkg:             pkg,
								sourceFileNodes: map[*ast.File]struct{}{},
							}
						}
						fileInfo := fileInfoByPath[outputPath]
						fileInfo.sourceFileNodes[fileNode] = struct{}{}
						fileInfo.objects = append(fileInfo.objects, object)
						return true
					}
				}
				return true
			})
		}
	}

	filesByPath := map[string]File{}
	for outputPath, fileInfo := range fileInfoByPath {
		file, fileErr := getFile(*fileInfo)
		if fileErr != nil {
			return nil, fileErr
		}
		filesByPath[outputPath] = file
	}
	return filesByPath, nil
}

// GetInterface searches the given package for the given interface and returns
// its text-template-friendly representation.
func GetInterface(pkg *packages.Package, ifaceName string) (File, error) {
	// Find the interface by name
	object := pkg.Types.Scope().Lookup(ifaceName)
	if object == nil {
		return File{}, fmt.Errorf("interface %s not found in package %s", ifaceName, pkg.Name)
	}

	// Find the file node containing this interface's definition.
	var ifaceFileNode *ast.File
	for _, fileNode := range pkg.Syntax {
		if pkg.Fset.File(fileNode.Pos()) == pkg.Fset.File(object.Pos()) {
			ifaceFileNode = fileNode
			break
		}
	}
	if ifaceFileNode == nil {
		return File{}, fmt.Errorf("declaration for interface %s not found in package %s's syntax trees", ifaceName, pkg.Name)
	}

	return getFile(fileInfo{
		pkg:             pkg,
		sourceFileNodes: map[*ast.File]struct{}{ifaceFileNode: {}},
		objects:         []types.Object{object},
	})
}

// These packages should be kept in sync with references in template.tmpl.
var defaultImports = map[string]bool{
	"sync/atomic": true,
	"testing":     true,
}

// getFile uses syntactic and type information about a file of mockable
// interfaces to construct a text-template-friendly representation of that file.
func getFile(fileInfo fileInfo) (File, error) {
	// Aggregate the source files' imports, along with their names (if renamed).
	var imports []Import
	for sourceFile := range fileInfo.sourceFileNodes {
		for _, fileImport := range sourceFile.Imports {
			imp := Import{
				Path: strings.Trim(fileImport.Path.Value, `"`),
			}
			if fileImport.Name != nil {
				imp.Name = fileImport.Name.Name
			}
			imports = append(imports, imp)
		}
	}

	// Every mock file includes imports required to implement the mock itself.
	for path := range defaultImports {
		imports = append(imports, Import{Path: path})
	}

	// If there are any conflicting imports (i.e. imports of different packages
	// with the same name originating from different source files), we need to
	// rename them to resolve the conflicts. To ensure the packages we choose to
	// rename are deterministic, we sort the packages.
	slices.SortFunc(imports, func(a, b Import) int {
		// Prioritize the default imports since the mock template assumes those
		// packages are not aliased.
		aDflt, bDflt := defaultImports[a.Path], defaultImports[b.Path]
		switch {
		case aDflt && !bDflt:
			return -1
		case !aDflt && bDflt:
			return 1
		}

		return cmp.Or(
			cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Name, b.Name),
		)
	})

	// Remove consecutive equal elements, in case the same import is included in
	// two source files.
	imports = slices.Compact(imports)

	// Finally, ensure uniqueness of local package names.
	packageTaken := map[string]bool{}
	for i := range imports {
		imp := &imports[i]
		for {
			localName := imp.LocalName()
			if !packageTaken[localName] {
				packageTaken[localName] = true
				break
			}
			name, incrementErr := incrementName(localName)
			if incrementErr != nil {
				return File{}, fmt.Errorf("resolving import conflict: %v", incrementErr)
			}
			imp.Name = name
		}
	}

	var (
		file      = File{Package: fileInfo.pkg.Name}
		qualifier = qualify(fileInfo.pkg.Types, imports, &file.Imports)
	)
	for _, object := range fileInfo.objects {
		iface, ifaceErr := getInterface(fileInfo, qualifier, object)
		if ifaceErr != nil {
			return File{}, ifaceErr
		}
		file.Interfaces = append(file.Interfaces, iface)
	}
	return file, nil
}

// incrementName adds one to the last number in the given name. If the name
// doesn't end in a number, incrementName appends "2".
func incrementName(name string) (string, error) {
	var (
		runes = []rune(name)
		cut   = 0
	)
	for i, r := range slices.Backward(runes) {
		if !unicode.IsDigit(r) {
			cut = i + 1
			break
		}
	}
	if cut == len(runes) {
		return name + "2", nil
	}
	alpha, numeric := runes[:cut], runes[cut:]
	n, parseErr := strconv.Atoi(string(numeric))
	if parseErr != nil {
		return "", fmt.Errorf("parsing numeric component of name: %v", parseErr)
	}
	return fmt.Sprintf("%v%v", string(alpha), n+1), nil
}

// getInterface uses syntactic and type information about an interface to
// construct a text-template-friendly representation of that interface.
func getInterface(fileInfo fileInfo, qualifier types.Qualifier, object types.Object) (Interface, error) {
	// Validate that the object is indeed an interface declaration.
	if _, isTypeName := object.(*types.TypeName); !isTypeName {
		return Interface{}, fmt.Errorf("%s is not a named/defined type", object.Name())
	}
	ifaceType, isInterface := object.Type().Underlying().(*types.Interface)
	if !isInterface {
		return Interface{}, fmt.Errorf("%s is not an interface type", object.Name())
	}

	// Make sure that none of the types involved in the interface's definition
	// were invalid/had errors.
	if !validateType(object.Type(), map[types.Type]bool{}) {
		return Interface{}, &TypeErrors{Errs: fileInfo.pkg.Errors}
	}

	// Begin assembling information about the interface.
	iface := Interface{Name: object.Name()}

	// Record type parameter list info.
	if ifaceNamed, ok := object.Type().(*types.Named); ok {
		typeParams := ifaceNamed.TypeParams()
		for i := range typeParams.Len() {
			typeParam := typeParams.At(i)
			iface.TypeParams = append(iface.TypeParams, TypeParam{
				Name:       typeParam.Obj().Name(),
				Constraint: types.TypeString(typeParam.Constraint(), qualifier),
			})
		}
	}

	// Iterate through each embedded interface's explicit methods.
	for _, ifaceType := range explodeInterface(ifaceType) {
		for i := range ifaceType.NumExplicitMethods() {
			methodObj := ifaceType.ExplicitMethod(i)
			method := Method{
				Name:     methodObj.Name(),
				srcIface: ifaceType.String(),
				pos:      methodObj.Pos(),
			}

			sig, ok := methodObj.Type().(*types.Signature)
			if !ok {
				return Interface{}, fmt.Errorf("%s is not a method signature", methodObj.Name())
			}

			// Keep track of the names and types of the parameters.
			paramsTuple := sig.Params()
			for j := 0; j < paramsTuple.Len(); j++ {
				paramObj := paramsTuple.At(j)
				param := Param{
					Name: paramObj.Name(),
					Type: types.TypeString(paramObj.Type(), qualifier),
				}
				method.Params = append(method.Params, param)
			}

			// Mark whether the last parameter is variadic.
			if len(method.Params) > 0 && sig.Variadic() {
				method.Params[len(method.Params)-1].Variadic = true
			}

			// Keep track of the names and types of the results.
			resultsTuple := sig.Results()
			for j := 0; j < resultsTuple.Len(); j++ {
				resultObj := resultsTuple.At(j)
				result := Result{
					Name: resultObj.Name(),
					Type: types.TypeString(resultObj.Type(), qualifier),
				}
				method.Results = append(method.Results, result)
			}

			if !slices.ContainsFunc(iface.Methods, method.Equal) {
				iface.Methods = append(iface.Methods, method)
			}
		}
	}

	// Preserve the original ordering of the methods.
	sort.Sort(iface.Methods)

	return iface, nil
}

// explodeInterface traverses an interface type, returning the original
// interface along with all transitively embedded interfaces.
func explodeInterface(iface *types.Interface) []*types.Interface {
	var (
		result    []*types.Interface
		workQueue = []*types.Interface{iface}
		visited   = map[string]bool{}
	)
	for len(workQueue) > 0 {
		current := workQueue[0]
		workQueue = workQueue[1:]
		currentID := current.String()
		if !visited[currentID] {
			visited[currentID] = true
			result = append(result, current)
			for i := range current.NumEmbeddeds() {
				switch embedded := current.EmbeddedType(i).(type) {
				case *types.Interface:
					workQueue = append(workQueue, embedded)
				case *types.Named:
					switch underlying := embedded.Underlying().(type) {
					case *types.Interface:
						workQueue = append(workQueue, underlying)
					}
				}
			}
		}
	}
	return result
}

func qualify(pkg *types.Package, imps []Import, usedImps *[]Import) types.Qualifier {
	return func(other *types.Package) string {
		// If the type is from this package, don't qualify it
		if pkg == other {
			return ""
		}

		// Search for the import statement for the package
		// that the type is from
		for _, imp := range imps {
			if other.Path() == imp.Path {
				// If the package was only imported for its
				// side-effects, skip over it
				if imp.Name == "_" {
					continue
				}

				// Keep track of the file imports that have actually
				// been used in this interface definition (de-duped)
				var found bool
				for _, usedImprt := range *usedImps {
					if imp.Path == usedImprt.Path {
						found = true
						break
					}
				}
				if !found {
					*usedImps = append(*usedImps, imp)
				}

				// If the package was brought into this package
				// in an unqualified manner, don't qualify it
				if imp.Name == "." {
					return ""
				}

				// If the package was renamed in the import
				// statement, return it's name
				if imp.Name != "" {
					return imp.Name
				}

				// Othewise, the package was not renamed,
				// so break out and return the package name
				break
			}
		}

		// We were unable to find an import statement in the original
		// file containing the interface that corresponds to the type.
		// This can happen if, for example, the interface embeds another
		// type from a different file/package. Add a corresponding import
		// to the list of used imports.
		// TODO: Because this is a new import that's not coming from the
		// original file, it could cause naming conflicts
		*usedImps = append(*usedImps, Import{
			Path: other.Path(),
		})
		return other.Name()
	}
}

func validateType(typ types.Type, visited map[types.Type]bool) bool {
	if visited[typ] {
		return true
	}
	visited[typ] = true

	switch t := typ.(type) {
	case nil:
		return true

	case *types.Basic:
		return t.Kind() != types.Invalid

	case *types.Array:
		return validateType(t.Elem(), visited)

	case *types.Slice:
		return validateType(t.Elem(), visited)

	case *types.Struct:
		for i := range t.NumFields() {
			if !validateType(t.Field(i).Type(), visited) {
				return false
			}
		}
		return true

	case *types.Pointer:
		return validateType(t.Elem(), visited)

	case *types.Tuple:
		for i := range t.Len() {
			if !validateType(t.At(i).Type(), visited) {
				return false
			}
		}
		return true

	case *types.Signature:
		return validateType(t.Params(), visited) &&
			validateType(t.Results(), visited)

	case *types.Interface:
		for i := range t.NumEmbeddeds() {
			if !validateType(t.EmbeddedType(i), visited) {
				return false
			}
		}
		for i := range t.NumMethods() {
			if !validateType(t.Method(i).Type(), visited) {
				return false
			}
		}
		return true

	case *types.Union:
		for i := range t.Len() {
			if !validateType(t.Term(i).Type(), visited) {
				return false
			}
		}
		return true

	case *types.Map:
		return validateType(t.Elem(), visited) &&
			validateType(t.Key(), visited)

	case *types.Chan:
		return validateType(t.Elem(), visited)

	case *types.Named:
		typeParams := t.TypeParams()
		for i := range typeParams.Len() {
			if !validateType(typeParams.At(i).Constraint(), visited) {
				return false
			}
		}
		return validateType(t.Underlying(), visited)

	default:
		// log.Printf("Unknown types.Type: %v", t)
		return true
	}
}
