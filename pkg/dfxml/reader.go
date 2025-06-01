package dfxml

import (
	"encoding/xml"
	"io"
)

// ReadFileObjects parses and returns all <fileobject> elements from the reader.
func ReadFileObjects(r io.Reader) ([]FileObject, error) {
	dec := xml.NewDecoder(r)
	var fileObjects []FileObject

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Look for start elements named "fileobject"
		if startElem, ok := tok.(xml.StartElement); ok && startElem.Name.Local == "fileobject" {
			var fo FileObject
			if err := dec.DecodeElement(&fo, &startElem); err != nil {
				return nil, err
			}
			fileObjects = append(fileObjects, fo)
		}
	}
	return fileObjects, nil
}
