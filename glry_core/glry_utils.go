package glry_core

import (
	gfcore "github.com/gloflow/gloflow/go/gf_core"
)

//-------------------------------------------------------------
// VALIDATE
func Validate(pInput interface{},
	pRuntime *Runtime) *gfcore.Gf_error {

	err := pRuntime.Validator.Struct(pInput)
	if err != nil {
		gErr := gfcore.Error__create("failed to validate HTTP input", 
			"verify__invalid_input_struct_error",
			map[string]interface{}{"input": pInput,},
			err, "glry_core", pRuntime.RuntimeSys)
		return gErr
	}
	
	return nil
}