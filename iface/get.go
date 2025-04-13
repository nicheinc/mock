package iface

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ifaceInfo represents syntactic and type information associated with an
// interface definition.
type ifaceInfo struct {
	pkg        *packages.Package
	fileNode   *ast.File
	object     types.Object
	outputPath string
}

// GetAllInterfaces searches the given packages for interfaces annotated with a
// "go:mock" directive, returning text-template-friendly representations.
func GetAllInterfaces(pkgs []*packages.Package) ([]Interface, error) {
	var allInfo []ifaceInfo
	for _, pkg := range pkgs {
		for _, fileNode := range pkg.Syntax {
			ast.Inspect(fileNode, func(node ast.Node) bool {
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
					outputPath := strings.TrimSpace(args)
					if outputPath == "" {
						inputPath := pkg.Fset.File(fileNode.Pos()).Name()
						outputPath = strings.TrimSuffix(inputPath, ".go") + "_mock.go"
					}
					for _, spec := range decl.Specs {
						// Only consider type declarations.
						spec, isType := spec.(*ast.TypeSpec)
						if !isType {
							continue
						}
						// Get the type info for this declaration.
						if object := pkg.TypesInfo.Defs[spec.Name]; object != nil {
							allInfo = append(allInfo, ifaceInfo{
								pkg:        pkg,
								fileNode:   fileNode,
								object:     object,
								outputPath: outputPath,
							})
							return true
						}
					}
				}
				return true
			})
		}
	}

	var ifaces []Interface
	for _, info := range allInfo {
		iface, ifaceErr := newInterface(info)
		if ifaceErr != nil {
			return nil, ifaceErr
		}
		ifaces = append(ifaces, iface)
	}
	return ifaces, nil
}

// GetInterface searches the given package for the given interface and returns
// its text-template-friendly representation. The provided outputPath, if
// nonempty, overrides the default output path.
func GetInterface(pkg *packages.Package, ifaceName string, outputPath string) (Interface, error) {
	// Find the interface by name
	ifaceObj := pkg.Types.Scope().Lookup(ifaceName)
	if ifaceObj == nil {
		return Interface{}, fmt.Errorf("interface %s not found in package %s", ifaceName, pkg.Name)
	}

	// Find the file node containing this interface's definition.
	var ifaceFileNode *ast.File
	for _, fileNode := range pkg.Syntax {
		if pkg.Fset.File(fileNode.Pos()) == pkg.Fset.File(ifaceObj.Pos()) {
			ifaceFileNode = fileNode
			break
		}
	}
	if ifaceFileNode == nil {
		return Interface{}, fmt.Errorf("declaration for interface %s not found in package %s's syntax trees", ifaceName, pkg.Name)
	}

	return newInterface(ifaceInfo{
		pkg:        pkg,
		fileNode:   ifaceFileNode,
		object:     ifaceObj,
		outputPath: outputPath,
	})
}

// newInterface uses syntactic and type information about an interface to
// construct a text-template-friendly representation of that interface.
func newInterface(info ifaceInfo) (Interface, error) {
	// Validate that the object is indeed an interface declaration.
	if _, isTypeName := info.object.(*types.TypeName); !isTypeName {
		return Interface{}, fmt.Errorf("%s is not a named/defined type", info.object.Name())
	}
	ifaceType, isInterface := info.object.Type().Underlying().(*types.Interface)
	if !isInterface {
		return Interface{}, fmt.Errorf("%s is not an interface type", info.object.Name())
	}

	// Make sure that none of the types involved in the
	// interface's definition were invalid/had errors
	if !ValidateType(info.object.Type()) {
		return Interface{}, &TypeErrors{Errs: info.pkg.Errors}
	}

	// Get the containing file's imports, along with their names (if renamed).
	var imps []Import
	for _, fileImp := range info.fileNode.Imports {
		imp := Import{
			Path: strings.Trim(fileImp.Path.Value, `"`),
		}
		if fileImp.Name != nil {
			imp.Name = fileImp.Name.Name
		}
		imps = append(imps, imp)
	}

	// Begin assembling information about the interface.
	iface := Interface{
		Package:    info.pkg.Name,
		Name:       info.object.Name(),
		OutputPath: info.outputPath,
	}
	qualifier := Qualify(info.pkg.Types, imps, &iface.Imports)

	// Record type parameter list info.
	if ifaceNamed, ok := info.object.Type().(*types.Named); ok {
		typeParams := ifaceNamed.TypeParams()
		for i := range typeParams.Len() {
			typeParam := typeParams.At(i)
			iface.TypeParams = append(iface.TypeParams, TypeParam{
				Name:       typeParam.Obj().Name(),
				Constraint: types.TypeString(typeParam.Constraint(), qualifier),
			})
		}
	}

	// Iterate through each embedded interface's explicit methods
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

			// Keep track of the names and types of the parameters
			paramsTuple := sig.Params()
			for j := 0; j < paramsTuple.Len(); j++ {
				paramObj := paramsTuple.At(j)
				param := Param{
					Name: paramObj.Name(),
					Type: types.TypeString(paramObj.Type(), qualifier),
				}
				method.Params = append(method.Params, param)
			}

			// Mark whether the last parameter is variadic
			if len(method.Params) > 0 && sig.Variadic() {
				method.Params[len(method.Params)-1].Variadic = true
			}

			// Keep track of the names and types of the results
			resultsTuple := sig.Results()
			for j := 0; j < resultsTuple.Len(); j++ {
				resultObj := resultsTuple.At(j)
				result := Result{
					Name: resultObj.Name(),
					Type: types.TypeString(resultObj.Type(), qualifier),
				}
				method.Results = append(method.Results, result)
			}

			iface.Methods = append(iface.Methods, method)
		}
	}

	// Preserve the original ordering of the methods
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

func Qualify(pkg *types.Package, imps []Import, usedImps *[]Import) types.Qualifier {
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

func ValidateType(typ types.Type) bool {
	return validateType(typ, map[types.Type]bool{})
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
