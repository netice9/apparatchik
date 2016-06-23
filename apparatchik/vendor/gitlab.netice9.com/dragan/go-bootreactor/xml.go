package bootreactor

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

func MustParseDisplayModel(src string) *DisplayModel {
	ret, err := ParseDisplayModel(src)
	if err != nil {
		panic(err)
	}
	return ret
}

func ParseDisplayModel(src string) (*DisplayModel, error) {
	reader := strings.NewReader(src)

	decoder := xml.NewDecoder(reader)

	stack := []*DisplayModel{}

	for {
		token, err := decoder.Token()
		// fmt.Println("token", token)

		if token == nil && err == io.EOF {
			var result *DisplayModel

			if len(stack) > 0 {
				result = stack[0]
			}
			return result, nil
		}

		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			model := &DisplayModel{
				Element:    t.Name.Local,
				Attributes: map[string]interface{}{},
			}
			for _, attribute := range t.Attr {
				if attribute.Name.Local == "id" {
					model.ID = attribute.Value
				} else if attribute.Name.Local == "reportEvents" {
					model.ReportEvents = strings.Split(attribute.Value, ",")
				} else {
					var value interface{} = attribute.Value
					if attribute.Name.Space == "bool" {
						value, err = strconv.ParseBool(attribute.Value)
						if err != nil {
							return nil, err
						}
					} else if attribute.Name.Space == "int" {
						value, err = strconv.Atoi(attribute.Value)
						if err != nil {
							return nil, err
						}
					}
					if attribute.Name.Local == "htmlID" {
						model.Attributes["id"] = value
					} else {
						model.Attributes[attribute.Name.Local] = value
					}
				}
			}
			if len(stack) != 0 {
				prev := stack[len(stack)-1]
				prev.Children = append(prev.Children, model)
			}
			stack = append(stack, model)

		case xml.CharData:
			if len(stack) != 0 {

				text := strings.TrimSpace(string(t))

				if text != "" {
					model := &DisplayModel{
						Text: &text,
					}
					prev := stack[len(stack)-1]
					prev.Children = append(prev.Children, model)
				}
			}
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}

}
