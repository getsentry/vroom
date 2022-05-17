package android

import (
	"strings"
	"testing"

	"github.com/getsentry/vroom/internal/testutil"
)

func assertEquals(t *testing.T, expected string, actual string) {
	if diff := testutil.Diff(expected, actual); diff != "" {
		t.Fatalf("expected \"%v\" but was \"%v\"", expected, actual)
	}
}

func assertFails(t *testing.T, err error, contains string) {
	if err == nil {
		t.Fatal("expected error to be non-nil")
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("expected error message to contain %q but it did not", err.Error())
	}
}

func TestParseBytecodeSignatureEmpty(t *testing.T) {
	_, _, err := ParseBytecodeSignature("")
	assertFails(t, err, "expected the character '('")
}

func TestParseBytecodeSignatureMissingReturn(t *testing.T) {
	_, _, err := ParseBytecodeSignature("()")
	assertFails(t, err, "expected a return type")
}

func TestParseBytecodeSignatureMultipleReturn(t *testing.T) {
	_, _, err := ParseBytecodeSignature("()ZI")
	assertFails(t, err, "did not end as expected")
}

func TestParseBytecodeSignatureMissingParameters(t *testing.T) {
	_, _, err := ParseBytecodeSignature("Z")
	assertFails(t, err, "expected the character '('")
}

func TestSimpleJavaTypeFromBytecodeTypeEmpty(t *testing.T) {
	_, err := SimpleJavaTypeFromBytecodeType("")
	assertFails(t, err, "must not be empty")
}

func TestSimpleJavaTypeFromBytecodeTypeInvalid(t *testing.T) {
	_, err := SimpleJavaTypeFromBytecodeType("Y")
	assertFails(t, err, "invalid descriptor type")
}

func TestSimpleJavaTypeFromBytecodeInvalidClass(t *testing.T) {
	_, err := SimpleJavaTypeFromBytecodeType("Lmissing.semicolon")
	assertFails(t, err, "invalid descriptor type, expected ';'")
}

func TestSimpleJavaTypeFromBytecodePrimitive(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("Z")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "boolean", javaType)
}

func TestSimpleJavaTypeFromBytecodePrimitiveArray(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("[I")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "int[]", javaType)
}

func TestSimpleJavaTypeFromBytecodeLang(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("Ljava/lang/String;")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "String", javaType)
}

func TestSimpleJavaTypeFromBytecodeLangArray(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("[Ljava/lang/String;")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "String[]", javaType)
}

func TestSimpleJavaTypeFromBytecodeClass(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("Landroid/app/Activity;")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "Activity", javaType)
}

func TestSimpleJavaTypeFromBytecodeClassArray(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("[Landroid/app/Activity;")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "Activity[]", javaType)
}

func TestSimpleJavaTypeFromBytecodeInnerClass(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("Lcom/google/common/util/concurrent/AbstractFuture$Listener;")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "AbstractFuture$Listener", javaType)
}

func TestSimpleJavaTypeFromBytecodeDoubleArray(t *testing.T) {
	javaType, err := SimpleJavaTypeFromBytecodeType("[[J")
	if err != nil {
		t.Fatal(err)
	}
	assertEquals(t, "long[][]", javaType)
}
