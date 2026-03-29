package main

import (
	"bufio"
	"io"
	"os"
	"strings"

	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newLmtdsCmd() *cobra.Command {
	var migrationsPath string
	var output string

	cmd := &cobra.Command{
		Use:   "generate db/migrations output.txt",
		Short: "Generate dbdiagram schema from migrations",
		Long:  `Parse every migration file of a given directory and every file of a subdirectory to generate the equivalent dbdiagram schema.`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return listFilesRecursively(migrationsPath)
		},
	}
	cmd.Flags().StringVarP(&migrationsPath, "migrations-path", "m", "database/migrations", "database migrations folder path")
	cmd.Flags().StringVarP(&output, "output", "o", "dbdigram.txt", "output file name, ex: dbdigram.txt")
	return cmd
}

func listFilesRecursively(migrationsPath string) error {
	return filepath.WalkDir(migrationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.Contains(path, "charset") || strings.Contains(path, "alterar") || strings.Contains(path, "trigger") {
			return nil
		}
		return parseFile(path)
	})
}

func parseFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewReader(file)

	extractedSchemaTypes := []*SchemaType{}
	parsingClass := false
	parsingUpMethod := false
	var className strings.Builder
	for {
		line, _, err := scanner.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		trimmedLine := strings.TrimSpace(string(line))
		if trimmedLine == "" {
			continue
		}

		if strings.HasPrefix(trimmedLine, "return new class") {
			parsingClass = true
			continue
		}

		if parsingClass == true {
			if strings.HasPrefix(trimmedLine, "public function up") {
				parsingUpMethod = true
				continue
			}

			if strings.HasPrefix(trimmedLine, "public function down") {
				continue
			}
		}

		if strings.HasPrefix(trimmedLine, "{") {
			continue
		}

		if parsingUpMethod == true {
			if strings.HasPrefix(trimmedLine, "Schema::create") {
				parsingString := false
				for _, char := range trimmedLine {
					if string(char) == "'" || string(char) == "\"" {
						if !parsingString {
							parsingString = true
							continue
						}
						break
					}
					if parsingString {
						className.WriteRune(char)
					}
				}
				continue
			}

			if strings.HasPrefix(trimmedLine, "}") {
				break
			}

			st, err := parseLine(trimmedLine)
			if err != nil {
				return err
			}
			extractedSchemaTypes = append(extractedSchemaTypes, st)
		}
	}

	var dbschemaStr strings.Builder

	fmt.Fprintf(&dbschemaStr, "Table %v {\n", className.String())

	for _, t := range extractedSchemaTypes {
		switch t.typeValue {
		case "increments":
			parseId(&dbschemaStr)
			continue
		case "unsignedInteger":
			parseUnsignedInteger(&dbschemaStr, t)
			continue
		case "integer":
			parseInteger(&dbschemaStr, t)
			continue
		case "string":
			parseString(&dbschemaStr, t)
			continue
		case "boolean":
			parseBoolean(&dbschemaStr, t)
			continue
		case "enum":
			parseEnum(&dbschemaStr, t)
			continue
		case "timestamp":
			parseTimestamp(&dbschemaStr, t)
			continue
		default:
			// fmt.Printf("    unknown typeValue: %v, nameValue: %v\n", t.typeValue, t.nameValue)
			continue
		}
	}
	fmt.Fprintf(&dbschemaStr, "}\n")

	outputPath := "dbdiagram.schema"

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", outputPath, err)
	}
	defer f.Close()

	if _, err = f.Write([]byte(dbschemaStr.String())); err != nil {
		return fmt.Errorf("error writing to file %s: %w", outputPath, err)
	}

	return nil
}

func parseLine(line string) (*SchemaType, error) {
	st := SchemaType{}
	parts := strings.Split(line, "->")[1:]
	if len(parts) == 0 {
		return &SchemaType{}, nil
	}
	newst, err := createSchemaType(parts[0])
	if err != nil {
		return &SchemaType{}, fmt.Errorf("error call to createSchemaType(token) with %v\n", newst)
	}

	st.typeValue = newst.typeValue
	st.nameValue = newst.nameValue

	for _, token := range parts[1:] {
		newst, err := createSchemaType(token)
		if err != nil {
			return &SchemaType{}, fmt.Errorf("error call to createSchemaType(token) with %v\n", token)
		}

		st.additional = append(st.additional, newst)
	}

	return &st, nil
}

type SchemaType struct {
	typeValue  string
	nameValue  string
	additional []*SchemaType
}

func createSchemaType(typeFuncStr string) (*SchemaType, error) {
	if len(typeFuncStr) < 3 {
		return &SchemaType{}, fmt.Errorf("createSchemaType expects a typeFuncStr with len greater than 2")
	}

	var typeValue strings.Builder
	var nameValue strings.Builder

	parsingNameValue := false

	i := 0
	for {
		char := string(typeFuncStr[i])

		if char == "(" {
			peekChar := string(typeFuncStr[i+1])
			if peekChar != "'" && peekChar != "\"" {
				break
			}
			parsingNameValue = true
			i += 2
			continue
		}

		if parsingNameValue && (char == "'" || char == "\"") {
			break
		}

		if parsingNameValue {
			nameValue.WriteString(char)
		} else {
			typeValue.WriteString(char)
		}

		i++
		if i >= len(string(typeFuncStr)) {
			break
		}
	}

	return &SchemaType{
		typeValue: typeValue.String(),
		nameValue: nameValue.String(),
	}, nil
}

func printSchemaType(st *SchemaType) {
	fmt.Printf("    SchemaType{typeValue: |%v|, nameValue: |%v|}\n", st.typeValue, st.nameValue)
	for _, a := range st.additional {
		fmt.Printf("        SchemaType{typeValue: |%v|, nameValue: |%v|}\n", a.typeValue, a.nameValue)
	}
}

func parseId(sb *strings.Builder) {
	(*sb).WriteString("   id integer [primary key]\n")
}

func parseUnsignedInteger(sb *strings.Builder, st *SchemaType) {
	fmt.Fprintf(sb, "   %v integer [not null]\n", st.nameValue)
}

func parseInteger(sb *strings.Builder, st *SchemaType) {
	fmt.Fprintf(sb, "   %v integer [not null]\n", st.nameValue)
}

func parseString(sb *strings.Builder, st *SchemaType) {
	fmt.Fprintf(sb, "   %v varchar\n", st.nameValue)
}

func parseBoolean(sb *strings.Builder, st *SchemaType) {
	fmt.Fprintf(sb, "   %v boolean\n", st.nameValue)
}

func parseEnum(sb *strings.Builder, st *SchemaType) {
	var metadata, comment strings.Builder
	hasComment := false
	isNullable := false

	for _, a := range st.additional {
		if a.typeValue == "comment" {
			hasComment = true
			comment.WriteString(a.nameValue)
			continue
		}
		if a.typeValue == "nullable" {
			isNullable = true
			continue
		}
	}

	fmt.Fprintf(sb, "   %v enum", st.nameValue)
	if !isNullable && hasComment {
		fmt.Fprintf(&metadata, " [not null, note: '%v']\n", comment.String())
	} else if hasComment {
		fmt.Fprintf(&metadata, " [note: '%v']\n", comment.String())
	} else {
		fmt.Fprintf(&metadata, "\n")
	}
	fmt.Fprintf(sb, "%v", metadata.String())
}

func parseTimestamp(sb *strings.Builder, st *SchemaType) {
	fmt.Fprintf(sb, "   %v timestamp", st.nameValue)
	isNullable := false
	for _, a := range st.additional {
		if a.typeValue == "nullable" {
			isNullable = true
			break
		}
	}
	if !isNullable {
		fmt.Fprintf(sb, " [not null]")
	}
	fmt.Fprintf(sb, "\n")
}
