package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	notification "example.com/spec-gen/example/notification"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type OneOfStrategy int

const (
	// always choose the first choice in one_of
	First = iota
	// randomly select a choice in one_of
	Random
	// interactively choose (note: pauses and waits for input at each point)
	Choose
)

type Config struct {
	oneOfStrategy OneOfStrategy
	rng           *rand.Rand // Use a dedicated RNG instance
}

// NewConfig creates a new Config with the specified strategy
func NewConfig(strategy OneOfStrategy) Config {
	// Create a properly seeded random number generator
	src := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(src)

	return Config{
		oneOfStrategy: strategy,
		rng:           rng,
	}
}

// FieldError wraps an error with a field path for better context
type FieldError struct {
	Path  []string
	Cause error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("error at path %s: %v", strings.Join(e.Path, "."), e.Cause)
}

func (e *FieldError) Unwrap() error {
	return e.Cause
}

// NewFieldError creates a new FieldError with field name added to path
func NewFieldError(fieldName string, path []string, err error) error {
	return &FieldError{
		Path:  append(append([]string{}, path...), fieldName),
		Cause: err,
	}
}

func main() {
	// Default strategy
	var strategy OneOfStrategy = Choose

	// Parse command line args to set strategy if provided
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "first":
			strategy = First
		case "random":
			strategy = Random
		case "choose":
			strategy = Choose
		}
	}

	// Create config with proper random initialization
	config := NewConfig(strategy)

	// Create an example of the main message
	msg := &notification.NotificationMessage{}
	if err := fillProtoWithExamples(msg, config, nil); err != nil {
		fmt.Printf("Error generating example: %v\n", err)
		os.Exit(1)
	}

	// Use protojson instead of standard json marshaling
	marshaler := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		AllowPartial:    true,
		UseProtoNames:   false, // Use JSON field names (default)
		UseEnumNumbers:  false, // Use enum string names
		EmitUnpopulated: false, // Only emit set fields
	}

	jsonBytes, err := marshaler.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Example spec:\n%s\n\n", string(jsonBytes))

	// Test roundtrip: JSON back to proto
	testRoundtrip(jsonBytes)
}

func testRoundtrip(jsonBytes []byte) {
	// Unmarshal JSON back to proto to verify it works
	unmarshaler := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	testMsg := &notification.NotificationMessage{}
	err := unmarshaler.Unmarshal(jsonBytes, testMsg)
	if err != nil {
		fmt.Printf("ERROR: Failed to unmarshal JSON back to proto: %v\n", err)
	} else {
		fmt.Println("SUCCESS: JSON can be unmarshaled back to proto!")
	}
}

// fillProtoWithExamples fills a proto message with example values
func fillProtoWithExamples(msg proto.Message, config Config, path []string) error {
	m := msg.ProtoReflect()
	return fillMessage(m, config, path)
}

// fillMessage fills a protoreflect.Message with example values
func fillMessage(m protoreflect.Message, config Config, path []string) error {
	// Get field descriptors
	fds := m.Descriptor().Fields()

	// Group fields by oneof
	oneofFields := make(map[protoreflect.OneofDescriptor][]protoreflect.FieldDescriptor)
	regularFields := make([]protoreflect.FieldDescriptor, 0)

	for i := 0; i < fds.Len(); i++ {
		fd := fds.Get(i)

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
		fieldPath := append(append([]string{}, path...), string(fd.Name()))
		if err := setExampleValue(m, fd, config, fieldPath); err != nil {
			return NewFieldError(string(fd.Name()), path, err)
		}
	}

	// Then, handle oneofs according to the strategy
	for oneof, fields := range oneofFields {
		if len(fields) > 0 {
			var fieldToUse protoreflect.FieldDescriptor

			switch config.oneOfStrategy {
			case First:
				fieldToUse = fields[0]
			case Random:
				fieldToUse = fields[config.rng.Intn(len(fields))] // Use the config's RNG
			case Choose:
				// Interactive selection
				fmt.Printf("Choose a field for oneof %s:\n", oneof.Name())
				for i, fd := range fields {
					fmt.Printf("[%d] %s\n", i, fd.Name())
				}

				var choice int
				fmt.Print("Enter choice: ")
				_, err := fmt.Scanf("%d", &choice)
				if err != nil || choice < 0 || choice >= len(fields) {
					return NewFieldError(string(oneof.Name()), path,
						fmt.Errorf("invalid selection for oneof: %v", err))
				}
				fieldToUse = fields[choice]
			}

			fieldPath := append(append([]string{}, path...), string(fieldToUse.Name()))
			if err := setExampleValue(m, fieldToUse, config, fieldPath); err != nil {
				return NewFieldError(string(fieldToUse.Name()), path, err)
			}
		}
	}

	return nil
}

func createExampleScalarValue(fd protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(true), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(42), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(42), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(42), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(42), nil
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(3.14), nil
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(3.14), nil
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(fmt.Sprintf("example_%s", fd.Name())), nil
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(fmt.Sprintf("example_%s_bytes", fd.Name()))), nil
	case protoreflect.EnumKind:
		enumValues := fd.Enum().Values()
		// fmt.Printf("%v:%v\n", fd.Name(), enumValues.Len())
		// proto does not serialize 0, so it must always be the error value!
		if enumValues.Len() > 1 {
			return protoreflect.ValueOfEnum(enumValues.Get(1).Number()), nil
		}
		return protoreflect.Value{}, fmt.Errorf("enum %s has no values", fd.Name())
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported scalar type %v for field %s", fd.Kind(), fd.Name())
	}
}

func createMapKey(keyFd protoreflect.FieldDescriptor) (protoreflect.MapKey, error) {
	switch keyFd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString("example_key").MapKey(), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(1).MapKey(), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(1).MapKey(), nil
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(true).MapKey(), nil
	default:
		return protoreflect.ValueOfString("").MapKey(),
			fmt.Errorf("unsupported map key type %v", keyFd.Kind())
	}
}

func handleListField(m protoreflect.Message, fd protoreflect.FieldDescriptor, config Config, path []string) error {
	list := m.NewField(fd).List()

	if fd.Kind() == protoreflect.MessageKind {
		// Message list
		val := list.NewElement()
		if msgVal, ok := val.Interface().(protoreflect.Message); ok {
			itemPath := append(path, "[0]")
			if err := fillMessage(msgVal, config, itemPath); err != nil {
				return fmt.Errorf("in list item: %w", err)
			}
		} else {
			return fmt.Errorf("failed to convert list element to message")
		}
		list.Append(val)
	} else {
		// Scalar list
		scalarVal, err := createExampleScalarValue(fd)
		if err != nil {
			return fmt.Errorf("creating scalar list value: %w", err)
		}
		list.Append(scalarVal)
	}

	m.Set(fd, protoreflect.ValueOfList(list))
	return nil
}

func handleMapField(m protoreflect.Message, fd protoreflect.FieldDescriptor, config Config, path []string) error {
	mapVal := m.NewField(fd).Map()

	// Create a key
	key, err := createMapKey(fd.MapKey())
	if err != nil {
		return fmt.Errorf("creating map key: %w", err)
	}

	// Create and fill the value
	val := mapVal.NewValue()
	if msgVal, ok := val.Interface().(protoreflect.Message); ok {
		keyStr := fmt.Sprintf("[%v]", key.String())
		mapItemPath := append(path, keyStr)
		if err := fillMessage(msgVal, config, mapItemPath); err != nil {
			return fmt.Errorf("in map value: %w", err)
		}
	} else if fd.MapValue().Kind() != protoreflect.MessageKind {
		// For scalar map values
		scalarVal, err := createExampleScalarValue(fd.MapValue())
		if err != nil {
			return fmt.Errorf("creating scalar map value: %w", err)
		}
		val = scalarVal
	} else {
		return fmt.Errorf("failed to handle map value type")
	}

	mapVal.Set(key, val)
	m.Set(fd, protoreflect.ValueOfMap(mapVal))
	return nil
}

func setExampleValue(m protoreflect.Message, fd protoreflect.FieldDescriptor, config Config, path []string) error {
	if fd.IsMap() {
		return handleMapField(m, fd, config, path)
	}

	if fd.IsList() {
		return handleListField(m, fd, config, path)
	}

	if fd.Kind() == protoreflect.MessageKind {
		// Regular message field
		msgVal := m.NewField(fd).Message()
		if err := fillMessage(msgVal, config, path); err != nil {
			return fmt.Errorf("in message field: %w", err)
		}
		m.Set(fd, protoreflect.ValueOfMessage(msgVal))
		return nil
	}

	// Regular scalar field
	val, err := createExampleScalarValue(fd)
	if err != nil {
		return fmt.Errorf("creating scalar value: %w", err)
	}
	m.Set(fd, val)
	return nil
}
