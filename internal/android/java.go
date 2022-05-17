package android

import (
	"fmt"
	"strings"

	"github.com/getsentry/vroom/internal/errorutil"
)

type bytecodeSignatureDecodingState int

const (
	parametersStart              bytecodeSignatureDecodingState = 0
	parameterTypeOrParametersEnd                                = 1
	parameterType                                               = 2
	returnType                                                  = 4
	End                                                         = 5
)

// Parses a bytecode signature into its parameters and return type.
// The returned parameters can be nil if the signature is unknown.
func ParseBytecodeSignature(signature string) (parameters []string, returnTypeString string, error error) {
	if signature == "Unknown" {
		return nil, "", nil
	}

	var state = parametersStart
	arrayPrefix := ""

	// i incrementation is handled on a case by case basis.
	for i := 0; i < len(signature); {
		r := signature[i]

		switch state {
		case parametersStart:
			if r != '(' {
				return nil, "", fmt.Errorf("java: %w: invalid descriptor, expected the character '(' but got %q in %q", errorutil.ErrDataIntegrity, r, signature)
			}
			state = parameterTypeOrParametersEnd
			i++

		case parameterTypeOrParametersEnd:
			if r == ')' {
				state = returnType
				i++
			} else {
				// Parameters end has been ruled out so check the same rune again expecting only a parameter type.
				state = parameterType
			}

		case parameterType, returnType:
			if r == '[' {
				arrayPrefix += "["
				// State stays the same.
			} else {
				var bytecodeType string
				if r == 'L' {
					semicolonIndex := strings.Index(signature[i:], ";")
					bytecodeType = signature[i : i+semicolonIndex+1]
					// i is incremented again at the end of the case.
					i += semicolonIndex
				} else {
					bytecodeType = string(r)
				}
				bytecodeType = arrayPrefix + bytecodeType
				if state == returnType {
					returnTypeString = bytecodeType
					state = End
				} else {
					parameters = append(parameters, bytecodeType)
					state = parameterTypeOrParametersEnd
				}
				arrayPrefix = ""
			}
			i++

		case End:
			return nil, "", fmt.Errorf("java: %w: invalid descriptor, did not end as expected: %q", errorutil.ErrDataIntegrity, signature)
		}
	}

	if state != End {
		var expectation string
		switch state {
		case parametersStart:
			expectation = "the character '('"
		case parameterTypeOrParametersEnd:
			expectation = "a parameter type or the character ')'"
		case parameterType:
			expectation = "a parameter type"
		case returnType:
			expectation = "a return type"
		}
		return nil, "", fmt.Errorf("java: %w: invalid descriptor, expected %v but got: %q", errorutil.ErrDataIntegrity, expectation, signature)
	}

	return
}

var primitiveDescriptorTypetoJava = map[uint8]string{
	'Z': "boolean",
	'B': "byte",
	'C': "char",
	'S': "short",
	'I': "int",
	'J': "long",
	'F': "float",
	'D': "double",
	'V': "void",
}

// Converts a "bytecode type" into a "simple java type": package names are removed and types are converted to be
// human-readable.
//
// For example:
// - "Landroid/app/Activity;" becomes "Activity"
// - "Ljava/lang/String;" becomes "String"
// - "Lcom/google/common/util/concurrent/AbstractFuture$Listener;" becomes "AbstractFuture$Listener"
// - "[Z" becomes "boolean[]"
// - "Z" becomes "boolean"
func SimpleJavaTypeFromBytecodeType(bytecodeType string) (javaType string, error error) {
	return javaTypeFromBytecodeType(bytecodeType, func(truncatedBytecodeClass string) string {
		lastSlashIndex := strings.LastIndex(truncatedBytecodeClass, "/")
		if lastSlashIndex == -1 {
			return truncatedBytecodeClass
		} else {
			return truncatedBytecodeClass[lastSlashIndex+1:]
		}
	})

}

// Converts a "bytecode type" into a "java type" which is meant to be more easily read by humans and contains all the
// same information.
//
// For example:
// - "Landroid/app/Activity;" becomes "android.app.Activity"
// - "Ljava/lang/String;" becomes "java.lang.String"
// - "Lcom/google/common/util/concurrent/AbstractFuture$Listener;" becomes "com.google.common.util.concurrent.AbstractFuture$Listener"
// - "[Z" becomes "boolean[]"
// - "Z" becomes "boolean"
func JavaTypeFromBytecodeType(bytecodeType string) (javaType string, error error) {
	return javaTypeFromBytecodeType(bytecodeType, func(truncatedBytecodeClass string) string {
		return strings.ReplaceAll(truncatedBytecodeClass, "/", ".")
	})
}

// For more information about the bytecode format see:
// http://ftp.magicsoftware.com/www/help/mg9/How/howto_java_vm_type_signatures.htm
func javaTypeFromBytecodeType(bytecodeType string, transform func(truncatedBytecodeClass string) string) (javaType string, error error) {
	if len(bytecodeType) == 0 {
		return "", fmt.Errorf("java: %w: invalid descriptor type, must not be empty", errorutil.ErrDataIntegrity)
	}
	firstR := bytecodeType[0]
	if javaType, found := primitiveDescriptorTypetoJava[firstR]; found {
		return javaType, nil
	}
	switch firstR {
	case 'L':
		descriptorLength := len(bytecodeType)
		lastR := bytecodeType[descriptorLength-1]
		if lastR != ';' {
			return "", fmt.Errorf("java: %w: invalid descriptor type, expected ';' character at the end of class name but was '%c' in %q", errorutil.ErrDataIntegrity, lastR, bytecodeType)
		}
		truncatedBytecodeClass := bytecodeType[1 : descriptorLength-1]
		return transform(truncatedBytecodeClass), nil
	case '[':
		arrayType, err := javaTypeFromBytecodeType(bytecodeType[1:], transform)
		return arrayType + "[]", err
	default:
		return "", fmt.Errorf("java: %w: invalid descriptor type: %q", errorutil.ErrDataIntegrity, bytecodeType)
	}
}
