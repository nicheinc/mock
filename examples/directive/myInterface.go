package directive

import (
	"fmt"
	"html/template"

	//lint:ignore ST1001 This tests dot import support.
	. "os"
	renamed "text/template"

	// Blank imports should not be included in mock files.
	_ "embed"

	"github.com/nicheinc/mock/examples/directive/internal"
)

// MyInterface is a sample interface with a large number of
// methods of different signatures.
//
//go:mock
type MyInterface interface {
	NoParamsOrReturn()
	UnnamedParam(string)
	UnnamedVariadicParam(...string)
	BlankParam(_ string)
	BlankVariadicParam(_ ...string)
	NamedParam(str string)
	NamedVariadicParam(strs ...string)
	SameTypeNamedParams(str1, str2 string)
	InternalTypeParam(internal internal.Internal)
	ImportedParam(tmpl template.Template)
	ImportedVariadicParam(tmpl ...template.Template)
	RenamedImportParam(tmpl renamed.Template)
	RenamedImportVariadicParam(tmpls ...renamed.Template)
	DotImportParam(file File)
	DotImportVariadicParam(files ...File)
	SelfReferentialParam(intf MyInterface)
	SelfReferentialVariadicParam(intf ...MyInterface)
	StructParam(obj struct{ num int })
	StructVariadicParam(objs ...struct{ num int })
	EmbeddedStructParam(obj struct{ int })
	EmbeddedStructVariadicParam(objs ...struct{ int })
	EmptyInterfaceParam(intf interface{})
	EmptyInterfaceVariadicParam(intf ...interface{})
	InterfaceParam(intf interface {
		MyFunc(num int) error
	})
	InterfaceVariadicParam(intf ...interface {
		MyFunc(num int) error
	})
	InterfaceVariadicFuncParam(intf interface {
		MyFunc(nums ...int) error
	})
	InterfaceVariadicFuncVariadicParam(intf ...interface {
		MyFunc(nums ...int) error
	})
	EmbeddedInterfaceParam(intf interface {
		fmt.Stringer
	})
	ChannelParam(chanParam chan int)
	MapParam(mapParam map[int]int)

	UnnamedReturn() error
	MultipleUnnamedReturn() (int, error)
	BlankReturn() (_ error)
	NamedReturn() (err error)
	SameTypeNamedReturn() (err1, err2 error)
	RenamedImportReturn() (tmpl renamed.Template)
	DotImportReturn() (file File)
	SelfReferentialReturn() (intf MyInterface)
	StructReturn() (obj struct{ num int })
	EmbeddedStructReturn() (obj struct{ int })
	EmptyInterfaceReturn() (intf interface{})
	InterfaceReturn() (intf interface {
		MyFunc(num int) error
	})
	InterfaceVariadicFuncReturn() (intf interface {
		MyFunc(nums ...int) error
	})
	EmbeddedInterfaceReturn() (intf interface {
		fmt.Stringer
	})
	ChannelReturn() chan int
	MapReturn() map[int]int
}
