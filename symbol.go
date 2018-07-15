package sym

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lunixbochs/struc"
	"github.com/pkg/errors"
)

// A Symbol is a PS1 symbol.
type Symbol struct {
	// Symbol header.
	Hdr *SymbolHeader
	// Symbol body.
	Body SymbolBody
}

// String returns the string representation of the symbol.
func (sym *Symbol) String() string {
	return fmt.Sprintf("%v %v", sym.Hdr, sym.Body)
}

// Size returns the size of the symbol in bytes.
func (sym *Symbol) Size() int {
	hdrSize := binary.Size(*sym.Hdr)
	bodySize := sym.Body.BodySize()
	return hdrSize + bodySize
}

// A SymbolHeader is a PS1 symbol header.
type SymbolHeader struct {
	// Address or value of symbol.
	Value uint32 `struc:"uint32,little"`
	// Symbol kind; specifies type of symbol body.
	Kind Kind `struc:"uint8,little"`
}

// String returns the string representation of the symbol header.
func (hdr *SymbolHeader) String() string {
	return fmt.Sprintf("$%08x %v", hdr.Value, hdr.Kind)
}

// SymbolBody is the sum-type of all symbol bodies.
type SymbolBody interface {
	// BodySize returns the size of the symbol body in bytes.
	BodySize() int
}

// parseSymbol parses and returns a PS1 symbol.
func parseSymbol(r io.Reader) (*Symbol, error) {
	// Parse symbol header.
	sym := &Symbol{}
	hdr, err := parseSymbolHeader(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	sym.Hdr = hdr

	// Parse symbol body.
	body, err := parseSymbolBody(r, hdr.Kind)
	if err != nil {
		return sym, errors.WithStack(err)
	}
	sym.Body = body
	return sym, nil
}

// parseSymbolHeader parses and returns a PS1 symbol header.
func parseSymbolHeader(r io.Reader) (*SymbolHeader, error) {
	hdr := &SymbolHeader{}
	if err := struc.Unpack(r, &hdr); err != nil {
		return nil, errors.WithStack(err)
	}
	return hdr, nil
}

// parseSymbolBody parses and returns a PS1 symbol body.
func parseSymbolBody(r io.Reader, kind Kind) (SymbolBody, error) {
	parse := func(body SymbolBody) (SymbolBody, error) {
		if err := struc.Unpack(r, body); err != nil {
			return nil, errors.WithStack(err)
		}
		return body, nil
	}
	switch kind {
	case KindName1:
		return parse(&Name1{})
	case KindName2:
		return parse(&Name2{})
	case KindDef:
		return parse(&Def{})
	case KindDef2:
		return parseDef2(r)
	case KindOverlay:
		return parse(&Overlay{})
	default:
		return nil, errors.Errorf("support for symbol kind 0x%02X not yet implemented", uint8(kind))
	}
}

// --- [ 0x01 ] ----------------------------------------------------------------

// A Name1 symbol specifies the name of a symbol.
//
// Value of the symbol header specifies associated address.
type Name1 struct {
	// Name length.
	NameLen uint8 `struc:"uint8,little,sizeof=Name"`
	// Symbol name,
	Name string
}

// String returns the string representation of the name symbol.
func (body *Name1) String() string {
	// $00000000 1 __RHS2_data_size
	return body.Name
}

// BodySize returns the size of the symbol body in bytes.
func (body *Name1) BodySize() int {
	return 1 + int(body.NameLen)
}

// --- [ 0x02 ] ----------------------------------------------------------------

// A Name2 symbol specifies the name of a symbol.
//
// Value of the symbol header specifies associated address.
type Name2 struct {
	// Name length.
	NameLen uint8 `struc:"uint8,little,sizeof=Name"`
	// Symbol name,
	Name string
}

// String returns the string representation of the name symbol.
func (body *Name2) String() string {
	// $80010000 2 printattribute
	return body.Name
}

// BodySize returns the size of the symbol body in bytes.
func (body *Name2) BodySize() int {
	return 1 + int(body.NameLen)
}

// --- [ 0x94 ] ----------------------------------------------------------------

// A Def symbol specifies the class, type, size and name of a definition.
//
// Value of the symbol header specifies TODO.
type Def struct {
	// Definition class.
	Class Class `struc:"uint16,little"`
	// Definition type.
	Type Type `struc:"uint16,little"`
	// Definition size.
	Size uint32 `struc:"uint32,little"`
	// Name length.
	NameLen uint8 `struc:"uint8,little,sizeof=Name"`
	// Definition name,
	Name string
}

// String returns the string representation of the definition symbol.
func (body *Def) String() string {
	// $00000000 94 Def class TPDEF type UCHAR size 0 name u_char
	return fmt.Sprintf("class %v type %v size %v name %v", body.Class, body.Type, body.Size, body.Name)
}

// BodySize returns the size of the symbol body in bytes.
func (body *Def) BodySize() int {
	return 2 + 2 + 4 + 1 + int(body.NameLen)
}

// --- [ 0x96 ] ----------------------------------------------------------------

// A Def2 symbol specifies the class, type, size, dimensions, tag and name of a
// definition.
//
// Value of the symbol header specifies TODO.
type Def2 struct {
	// Definition class.
	Class Class `struc:"uint16,little"`
	// Definition type.
	Type Type `struc:"uint16,little"`
	// Definition size.
	Size uint32 `struc:"uint32,little"`
	// Dimensions
	Dims []Dimensions
	// Tag length.
	TagLen uint8 `struc:"uint8,little,sizeof=Tag"`
	// Definition tag,
	Tag string
	// Name length.
	NameLen uint8 `struc:"uint8,little,sizeof=Name"`
	// Definition name,
	Name string
}

// String returns the string representation of the definition symbol.
func (body *Def2) String() string {
	// $00000000 96 Def2 class MOS type ARY INT size 4 dims 1 1 tag  name r
	var dd []string
	for _, dims := range body.Dims {
		dd = append(dd, dims.String())
	}
	return fmt.Sprintf("class %v type %v size %v dims %v tag %v name %v", body.Class, body.Type, body.Size, strings.Join(dd, " "), body.Tag, body.Name)
}

// BodySize returns the size of the symbol body in bytes.
func (body *Def2) BodySize() int {
	dimsLen := 0
	for _, dims := range body.Dims {
		dimsLen += 2 * len(dims)
	}
	return 2 + 2 + 4 + dimsLen + 1 + int(body.TagLen) + 1 + int(body.NameLen)
}

// parseDef2 parses the body of a Def2 symbol.
func parseDef2(r io.Reader) (SymbolBody, error) {
	body := &Def2{}
	// Class
	if err := binary.Read(r, binary.LittleEndian, &body.Class); err != nil {
		return nil, errors.WithStack(err)
	}
	// Type
	if err := binary.Read(r, binary.LittleEndian, &body.Type); err != nil {
		return nil, errors.WithStack(err)
	}
	// Size
	if err := binary.Read(r, binary.LittleEndian, &body.Size); err != nil {
		return nil, errors.WithStack(err)
	}
	// Dims
	narray := 0
	for _, mod := range body.Type.mods() {
		// ARY
		if mod == 0x3 {
			narray++
		}
	}
	if narray == 0 {
		narray = 1
	}
	for i := 0; i < narray; i++ {
		var dims Dimensions
		if err := struc.Unpack(r, &dims); err != nil {
			return nil, errors.WithStack(err)
		}
		body.Dims = append(body.Dims, dims)
	}
	// Tag
	if err := binary.Read(r, binary.LittleEndian, &body.TagLen); err != nil {
		return nil, errors.WithStack(err)
	}
	if body.TagLen > 0 {
		buf := make([]byte, body.TagLen)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, errors.WithStack(err)
		}
		body.Tag = string(buf)
	}
	// Name
	if err := binary.Read(r, binary.LittleEndian, &body.NameLen); err != nil {
		return nil, errors.WithStack(err)
	}
	if body.NameLen > 0 {
		buf := make([]byte, body.NameLen)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, errors.WithStack(err)
		}
		body.Name = string(buf)
	}
	return body, nil
}

// --- [ 0x98 ] ----------------------------------------------------------------

// An Overlay symbol specifies the length and id of an file overlay (e.g. a
// shared library).
//
// Value of the symbol header specifies the base address at which the overlay is
// loaded.
type Overlay struct {
	// Overlay length in bytes.
	Length uint32 `struc:"uint32,little"`
	// Overlay ID.
	ID uint32 `struc:"uint32,little"`
}

// String returns the string representation of the overlay symbol.
func (body *Overlay) String() string {
	// $800b031c overlay length $000009e4 id $4
	return fmt.Sprintf("length $%08x id $%x", body.Length, body.ID)
}

// BodySize returns the size of the symbol body in bytes.
func (body *Overlay) BodySize() int {
	return 4 + 4
}

// ### [ Helper functions ] ####################################################

// Dimensions specifies array dimensions.
type Dimensions []uint16

func (dims *Dimensions) Pack(p []byte, opt *struc.Options) (int, error) {
	panic("not yet implemented")
}

func (dims *Dimensions) Unpack(r io.Reader, length int, opt *struc.Options) error {
	// TODO: figure out how to parse Dims of ARY ARY; e.g.
	//    000dc0: $00000000 96 Def2 class MOS type ARY ARY SHORT size 18 dims 2 3 3 tag  name m
	for {
		var dim uint16
		if err := binary.Read(r, binary.LittleEndian, &dim); err != nil {
			if errors.Cause(err) == io.EOF {
				return errors.WithStack(io.ErrUnexpectedEOF)
			}
			return errors.WithStack(err)
		}
		*dims = append(*dims, dim)
		if dim == 0 {
			break
		}
	}
	return nil
}

func (dims *Dimensions) Size(opt *struc.Options) int {
	return 2 * len(*dims)
}

func (dims Dimensions) String() string {
	var ds []string
	for _, dim := range dims {
		if dim == 0 {
			break
		}
		d := strconv.Itoa(int(dim))
		ds = append(ds, d)
	}
	if len(ds) == 0 {
		return "0"
	}
	return strings.Join(ds, " ")
}