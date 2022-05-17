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

func ExtractPackageNameAndSimpleMethodNameFromAndroidMethod(method *AndroidMethod) (string, string, error) {
	fullMethodName, err := FullMethodNameFromAndroidMethod(method)

	if err != nil {
		return "", "", err
	}

	packageName := packageNameFromAndroidMethod(method)

	return packageName, StripPackageNameFromFullMethodName(fullMethodName, packageName), nil
}

func FullMethodNameFromAndroidMethod(method *AndroidMethod) (string, error) {
	convertedSignature, err := ConvertedSignatureFromBytecodeSignature(method.Signature)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(method.ClassName)
	// "<init>" refers to the constructor in which case it's more readable to omit the method name. Note the method name
	// can also be a static initializer "<clinit>" but I don't know of any better ways to represent it so leaving as is.
	if method.Name != "<init>" {
		builder.WriteRune('.')
		builder.WriteString(method.Name)
	}
	builder.WriteString(convertedSignature)

	return builder.String(), nil
}

func packageNameFromAndroidMethod(method *AndroidMethod) string {
	index := strings.LastIndex(method.ClassName, ".")

	if index == -1 {
		return method.ClassName
	}

	return method.ClassName[:index]
}
