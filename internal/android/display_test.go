package android

import (
	"testing"
)

func TestNoParametersVoidReturnConvertedSignatureFromBytecodeSignature(t *testing.T) {
	javaSignature, err := ConvertedSignatureFromBytecodeSignature("()V")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "()", javaSignature)
}

func TestNoParametersConvertedSignatureFromBytecodeSignature(t *testing.T) {
	javaSignature, err := ConvertedSignatureFromBytecodeSignature("()F")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "(): float", javaSignature)
}

func TestVoidReturnConvertedSignatureFromBytecodeSignature(t *testing.T) {
	javaSignature, err := ConvertedSignatureFromBytecodeSignature("(Z)V")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "(boolean)", javaSignature)
}

func TestMultiParameterConvertedSignatureFromBytecodeSignature(t *testing.T) {
	javaType, err := ConvertedSignatureFromBytecodeSignature("(BLjava/lang/String;Landroid/view/View;)V")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "(byte, String, View)", javaType)
}

func TestArrayParameterArrayReturnConvertedSignatureFromBytecodeSignature(t *testing.T) {
	javaType, err := ConvertedSignatureFromBytecodeSignature("([Ljava/lang/String;[[B)[I")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "(String[], byte[][]): int[]", javaType)
}

func TestSingleParameterVoidReturnConvertedSignatureFromBytecodeSignature(t *testing.T) {
	javaType, err := ConvertedSignatureFromBytecodeSignature("(B)V")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "(byte)", javaType)
}

func TestUnknownConvertedSignatureFromBytecodeSignature(t *testing.T) {
	javaType, err := ConvertedSignatureFromBytecodeSignature("Unknown")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "", javaType)
}
