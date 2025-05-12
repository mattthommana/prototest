package main

import (
	"encoding/json"
	"fmt"

	notification "example.com/spec-gen/github.com/example/notification"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func main() {
	// Create an example of the main message
	msg := &notification.NotificationMessage{}
	fillProtoWithExamples(msg)

	// Convert to JSON
	jsonBytes, _ := json.MarshalIndent(msg, "", "  ")
	fmt.Printf("Example spec:\n%s\n\n", string(jsonBytes))
}

// fillProtoWithExamples fills a proto message with example values
func fillProtoWithExamples(msg proto.Message) {
	m := msg.ProtoReflect()
	fillMessage(m)
}

// fillMessage fills a protoreflect.Message with example values
func fillMessage(m protoreflect.Message) {
	// Get field descriptors
	fds := m.Descriptor().Fields()

	// Group fields by oneof
	oneofFields := make(map[protoreflect.OneofDescriptor][]protoreflect.FieldDescriptor)
	regularFields := make([]protoreflect.FieldDescriptor, 0)

	for i := 0; i < fds.Len(); i++ {
		fd := fds.Get(i)
		fmt.Printf("%v\n", fd.Name())

		if oneof := fd.ContainingOneof(); oneof != nil {
			// This field belongs to a oneof
			oneofFields[oneof] = append(oneofFields[oneof], fd)
		} else {
			// Regular field
			regularFields = append(regularFields, fd)
		}
	}

	// First, set all regular fields
	for _, fd := range regularFields {
		setExampleValue(m, fd)
	}

	// Then, for each oneof, pick the first option
	for _, fields := range oneofFields {
		if len(fields) > 0 {
			// Use the first field of each oneof
			setExampleValue(m, fields[0])
		}
	}
}

// setExampleValue sets an example value for a field
func setExampleValue(m protoreflect.Message, fd protoreflect.FieldDescriptor) {
	if fd.IsList() && fd.Kind() != protoreflect.MessageKind {
		list := m.NewField(fd).List()

		// Create an example value based on field type
		var val protoreflect.Value
		switch fd.Kind() {
		case protoreflect.BoolKind:
			val = protoreflect.ValueOfBool(true)
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			val = protoreflect.ValueOfInt32(42)
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			val = protoreflect.ValueOfInt64(42)
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			val = protoreflect.ValueOfUint32(42)
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			val = protoreflect.ValueOfUint64(42)
		case protoreflect.FloatKind:
			val = protoreflect.ValueOfFloat32(3.14)
		case protoreflect.DoubleKind:
			val = protoreflect.ValueOfFloat64(3.14)
		case protoreflect.StringKind:
			val = protoreflect.ValueOfString(fmt.Sprintf("example_%s", fd.Name()))
		case protoreflect.BytesKind:
			val = protoreflect.ValueOfBytes([]byte(fmt.Sprintf("example_%s_bytes", fd.Name())))
		case protoreflect.EnumKind:
			enumValues := fd.Enum().Values()
			if enumValues.Len() > 0 {
				val = protoreflect.ValueOfEnum(enumValues.Get(0).Number())
			}
		}

		// Add to list and set field
		list.Append(val)
		m.Set(fd, protoreflect.ValueOfList(list))
		return
	}

	switch fd.Kind() {
	case protoreflect.BoolKind:
		m.Set(fd, protoreflect.ValueOfBool(true))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		m.Set(fd, protoreflect.ValueOfInt32(42))
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		m.Set(fd, protoreflect.ValueOfInt64(42))
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		m.Set(fd, protoreflect.ValueOfUint32(42))
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		m.Set(fd, protoreflect.ValueOfUint64(42))
	case protoreflect.FloatKind:
		m.Set(fd, protoreflect.ValueOfFloat32(3.14))
	case protoreflect.DoubleKind:
		m.Set(fd, protoreflect.ValueOfFloat64(3.14))
	case protoreflect.StringKind:
		m.Set(fd, protoreflect.ValueOfString(fmt.Sprintf("example_%s", fd.Name())))
	case protoreflect.BytesKind:
		m.Set(fd, protoreflect.ValueOfBytes([]byte(fmt.Sprintf("example_%s_bytes", fd.Name()))))
	case protoreflect.EnumKind:
		// Use the first enum value by default
		enumValues := fd.Enum().Values()
		if enumValues.Len() > 0 {
			m.Set(fd, protoreflect.ValueOfEnum(enumValues.Get(0).Number()))
		}
	case protoreflect.MessageKind:
		if fd.IsMap() {
			// Handle maps
			mapVal := m.NewField(fd).Map()

			// Create a key based on the key kind
			var key protoreflect.MapKey
			keyFd := fd.MapKey()
			switch keyFd.Kind() {
			case protoreflect.StringKind:
				key = protoreflect.ValueOfString("example_key").MapKey()
			case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
				protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
				key = protoreflect.ValueOfInt64(1).MapKey()
			case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
				protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
				key = protoreflect.ValueOfUint64(1).MapKey()
			case protoreflect.BoolKind:
				key = protoreflect.ValueOfBool(true).MapKey()
			default:
				// Skip maps with unsupported key types
				return
			}

			// Create and fill the map value
			val := mapVal.NewValue()
			if msgVal, ok := val.Interface().(protoreflect.Message); ok {
				fillMessage(msgVal)
			}

			mapVal.Set(key, val)
			m.Set(fd, protoreflect.ValueOfMap(mapVal))
		} else if fd.IsList() {
			// Handle repeated message fields
			list := m.NewField(fd).List()
			val := list.NewElement()

			if msgVal, ok := val.Interface().(protoreflect.Message); ok {
				fillMessage(msgVal)
			}

			list.Append(val)
			m.Set(fd, protoreflect.ValueOfList(list))
		} else {
			// Regular message field
			msgVal := m.NewField(fd).Message()
			fillMessage(msgVal)
			m.Set(fd, protoreflect.ValueOfMessage(msgVal))
		}
	}
}
