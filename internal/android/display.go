package android

import (
	"strings"
)

// Returns a Java signature simplified for display in call trees. Package names are truncated and the return type
// follows a Kotlin-esque format, see tests for examples.
func ConvertedSignatureFromBytecodeSignature(bytecodeSignature string) (string, error) {
	if bytecodeSignature == "Unknown" {
		return "", nil
	}

	bytecodeParameters, bytecodeReturnType, err := ParseBytecodeSignature(bytecodeSignature)
	if err != nil {
		return "", err
	}

	convertedSignature := "("
	for _, bytecodeParameter := range bytecodeParameters {
		convertedParameter, paramErr := SimpleJavaTypeFromBytecodeType(bytecodeParameter)
		if paramErr != nil {
			return "", paramErr
		}
		convertedSignature = convertedSignature + convertedParameter + ", "
	}
	convertedSignature = strings.TrimSuffix(convertedSignature, ", ") + ")"

	convertedReturnType, err := SimpleJavaTypeFromBytecodeType(bytecodeReturnType)
	if err != nil {
		return "", err
	}
	if convertedReturnType != "void" {
		convertedSignature = convertedSignature + ": " + convertedReturnType
	}

	return convertedSignature, nil
}

func StripPackageNameFromFullMethodName(s, p string) string {
	return strings.TrimPrefix(s, p+".")
}
