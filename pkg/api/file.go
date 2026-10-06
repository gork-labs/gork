package api

import "reflect"

// File is one file part of a multipart/form-data request body.
//
// A Body struct with a File or []File field makes the body multipart/form-data.
// The gork tag of the field gives the name of the form field.
type File struct {
	// Name is the file name from the Content-Disposition header of the part.
	Name string
	// ContentType is the Content-Type header of the part.
	ContentType string
	Data        []byte
}

var (
	fileType        = reflect.TypeOf(File{})
	fileSliceType   = reflect.TypeOf([]File{})
	stringSliceType = reflect.TypeOf([]string{})
)

// isMultipartBody reports whether a Body struct has a File or []File field.
func isMultipartBody(bodyType reflect.Type) bool {
	if bodyType.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < bodyType.NumField(); i++ {
		if isFileType(bodyType.Field(i).Type) {
			return true
		}
	}
	return false
}

func isFileType(t reflect.Type) bool {
	return t == fileType || t == fileSliceType
}

func fileSchema() *Schema {
	return &Schema{Type: "string", ContentMediaType: "application/octet-stream"}
}
