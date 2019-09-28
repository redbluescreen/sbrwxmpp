// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmlstream

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

var NoMoreChildrenError = errors.New("no more children")

func NewStream(r io.Reader) (*ElementStream, error) {
	rd := xml.NewDecoder(r)
	for {
		tok, err := rd.Token()
		if err != nil {
			return nil, err
		}
		switch tok.(type) {
		case xml.StartElement:
			return &ElementStream{rd: rd}, nil
		}
	}
}

type Element struct {
	Name     xml.Name
	Attr     []xml.Attr
	Children []Element
	Text     string
}

func (e *Element) GetAttr(name string) string {
	for _, attr := range e.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func (e *Element) SetAttr(name string, value string) {
	for i, attr := range e.Attr {
		if attr.Name.Local == name {
			attr.Value = value
			e.Attr[i] = attr
			return
		}
	}
	e.Attr = append(e.Attr, xml.Attr{
		Name:  xml.Name{Local: name},
		Value: value,
	})
}

func (e *Element) GetChild(name string) (Element, bool) {
	for _, child := range e.Children {
		if child.Name.Local == name {
			return child, true
		}
	}
	return Element{}, false
}

func (e *Element) AsString() string {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	e.writeToEncoder(enc)
	enc.Flush()
	return buf.String()
}

func (e *Element) writeToEncoder(enc *xml.Encoder) error {
	attrs := make([]xml.Attr, 0)
	// Otherwise we would get duplicate xmlns attribute because Name.Space will
	// turn into xmlns too
	for _, attr := range e.Attr {
		if attr.Name.Space != "" || attr.Name.Local != "xmlns" {
			attrs = append(attrs, attr)
		}
	}
	err := enc.EncodeToken(xml.StartElement{
		Name: e.Name,
		Attr: attrs,
	})
	if err != nil {
		return err
	}
	for _, child := range e.Children {
		err = child.writeToEncoder(enc)
		if err != nil {
			return err
		}
	}
	if e.Text != "" {
		err = enc.EncodeToken(xml.CharData([]byte(e.Text)))
		if err != nil {
			return err
		}
	}
	return enc.EncodeToken(xml.EndElement{
		Name: e.Name,
	})
}

func (e *Element) readFromDecoder(d *xml.Decoder) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch v := tok.(type) {
		case xml.StartElement:
			el := Element{
				Name: v.Name,
				Attr: v.Attr,
			}
			err := el.readFromDecoder(d)
			if err != nil {
				return err
			}
			e.Children = append(e.Children, el)
		case xml.CharData:
			e.Text = string(v)
		case xml.EndElement:
			return nil
		}
	}
}

type ElementStream struct {
	xml.StartElement
	rd    *xml.Decoder
	ended bool
}

func (s *ElementStream) NextChild() (Element, error) {
	if s.ended {
		return Element{}, NoMoreChildrenError
	}
	for {
		tok, err := s.rd.Token()
		if err != nil {
			return Element{}, err
		}
		switch v := tok.(type) {
		case xml.StartElement:
			el := Element{
				Name: v.Name,
				Attr: v.Attr,
			}
			err := el.readFromDecoder(s.rd)
			return el, err
		case xml.EndElement:
			s.ended = true
			return Element{}, NoMoreChildrenError
		}
	}
}
