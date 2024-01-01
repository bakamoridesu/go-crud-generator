package main

import (
	"fmt"
	"regexp"
	"strings"
)

type FieldInfo struct {
	Name string
	Type string
}

type StructInfo struct {
	Name    string
	Fields  []FieldInfo
	IsMain  bool
	IDField string
}

var structMap = make(map[string]StructInfo)
var packageDefinition string // Define a global variable to store package definition

func generateJSONNotation(structString string) (string, error) {
	// Remove the package definition and trim whitespace
	structString = removePackageDefinition(structString)
	structString = strings.TrimSpace(structString)

	finalOutput := strings.TrimSpace(packageDefinition) + "\n\n" // Append package definition

	// Append necessary import statements
	finalOutput += "import (\n"
	finalOutput += "\t\"encoding/json\"\n"
	finalOutput += "\t\"net/http\"\n"
	finalOutput += "\t\"os\"\n"
	finalOutput += "\t\"strings\"\n"
	finalOutput += ")"

	// Regular expression to match multiple struct definitions in the input string
	re := regexp.MustCompile(`type\s+(\w+)\s+struct\s+{([^}]*)}`)
	matches := re.FindAllStringSubmatch(structString, -1)

	for _, match := range matches {
		if len(match) != 3 {
			return "", fmt.Errorf("invalid struct format")
		}

		structName := match[1]
		fieldString := match[2]

		fieldMatches := regexp.MustCompile(`(\w+)\s+(\w+)`).FindAllStringSubmatch(fieldString, -1)

		var fields []FieldInfo
		isMain := false

		for _, fieldMatch := range fieldMatches {
			fieldName := fieldMatch[1]
			fieldType := fieldMatch[2]

			field := FieldInfo{Name: fieldName, Type: fieldType}
			fields = append(fields, field)

			if isMainType(fieldName) {
				isMain = true
			}
		}

		structInfo := StructInfo{Name: structName, Fields: fields, IsMain: isMain}
		structMap[structName] = structInfo
	}

	var resultStrings []string

	resultStrings = append(resultStrings, finalOutput)
	// Generate JSON notation for each struct
	for _, structInfo := range structMap {
		var jsonFields []string
		for _, field := range structInfo.Fields {
			jsonFields = append(jsonFields, fmt.Sprintf("%s %s `json:\"%s\"`", field.Name, field.Type, strings.ToLower(field.Name)))
		}

		jsonStruct := fmt.Sprintf("type %s struct {\n\t%s\n}", structInfo.Name, strings.Join(jsonFields, "\n\t"))
		resultStrings = append(resultStrings, jsonStruct)
	}

	// Generate CRUD operations for the main struct
	mainStruct := getMainStruct(structMap)
	if mainStruct != nil {
		resultStrings = append(resultStrings, generateCRUDOperations(*mainStruct))
	}

	return strings.Join(resultStrings, "\n\n"), nil
}

func getMainStruct(structMap map[string]StructInfo) *StructInfo {
	for _, structInfo := range structMap {
		if structInfo.IsMain {
			return &structInfo
		}
	}
	return nil
}

func findIDField(fields []FieldInfo) string {
	for _, field := range fields {
		if isMainType(field.Name) {
			return field.Name
		}
	}
	return "" // or handle the case where 'id' field isn't found in the structure
}

func isMainType(fieldName string) bool {
	fieldName = strings.ToLower(fieldName)
	return strings.HasSuffix(fieldName, "id")
}

func removePackageDefinition(structString string) string {
	// Regular expression to match the package definition
	re := regexp.MustCompile(`package\s+\w+\s+`)
	// Find the package definition
	packageDef := re.FindString(structString)
	// Store the package definition in the global variable
	packageDefinition = packageDef
	// Remove the package definition from the structString
	modifiedStructString := re.ReplaceAllString(structString, "")
	return modifiedStructString
}

func generateCRUDOperations(mainStruct StructInfo) string {
	mainStruct.IDField = findIDField(mainStruct.Fields)
	dataMapDeclaration := "var dataMap = make(map[string]" + mainStruct.Name + ")\n"

	loadFunc := generateLoadFromJSONFunction(mainStruct.Name)
	saveFunc := generateSaveToJSONFunction(mainStruct.Name)

	createHandler := generateHTTPCreateHandler(mainStruct)
	readHandler := generateHTTPReadHandler(mainStruct)
	updateHandler := generateHTTPUpdateHandler(mainStruct)
	deleteHandler := generateHTTPDeleteHandler(mainStruct)

	loadAndSaveFuncs := "// Load and Save functions for " + mainStruct.Name + " data\n" + loadFunc + "\n\n" + saveFunc

	return dataMapDeclaration + "\n\n" + createHandler + "\n\n" + readHandler + "\n\n" + updateHandler + "\n\n" + deleteHandler + "\n\n" + loadAndSaveFuncs
}

func generateLoadFromJSONFunction(structName string) string {
	loadFunc := "func loadFromJSONFile(filename string) error {\n"
	loadFunc += "\tjsonData, err := os.ReadFile(filename)\n"
	loadFunc += "\tif err != nil {\n"
	loadFunc += "\t\treturn err\n"
	loadFunc += "\t}\n\n"
	loadFunc += "\terr = json.Unmarshal(jsonData, &dataMap)\n"
	loadFunc += "\tif err != nil {\n"
	loadFunc += "\t\treturn err\n"
	loadFunc += "\t}\n\n"
	loadFunc += "\treturn nil\n"
	loadFunc += "}\n"

	return loadFunc
}

func generateSaveToJSONFunction(structName string) string {
	saveFunc := "func saveToJSONFile(filename string) error {\n"
	saveFunc += "\tjsonData, err := json.MarshalIndent(dataMap, \"\", \"  \")\n"
	saveFunc += "\tif err != nil {\n"
	saveFunc += "\t\treturn err\n"
	saveFunc += "\t}\n\n"
	saveFunc += "\terr := os.WriteFile(filename, jsonData, 0644)\n"
	saveFunc += "\tif err != nil {\n"
	saveFunc += "\t\treturn err\n"
	saveFunc += "\t}\n\n"
	saveFunc += "\treturn nil\n"
	saveFunc += "}\n"

	return saveFunc
}

func generateHTTPCreateHandler(mainStruct StructInfo) string {
	createLogic := "func Create" + mainStruct.Name + "(w http.ResponseWriter, r *http.Request) {\n"
	createLogic += "\tif r.Method != http.MethodPost {\n"
	createLogic += "\t\tw.WriteHeader(http.StatusMethodNotAllowed)\n"
	createLogic += "\t\treturn\n"
	createLogic += "\t}\n\n"
	createLogic += "\tvar newData " + mainStruct.Name + "\n"
	createLogic += "\terr := json.NewDecoder(r.Body).Decode(&newData)\n"
	createLogic += "\tif err != nil {\n"
	createLogic += "\t\tw.WriteHeader(http.StatusBadRequest)\n"
	createLogic += "\t\treturn\n"
	createLogic += "\t}\n\n"
	createLogic += "\t// Code to save the newData to the dataMap\n"
	createLogic += "\tdataMap[newData." + mainStruct.IDField + "] = newData\n"
	createLogic += "\terr = saveToJSONFile(\"" + strings.ToLower(mainStruct.Name) + ".json\")\n\n"
	createLogic += "\tif err != nil {\n"
	createLogic += "\t\tw.WriteHeader(http.StatusInternalServerError)\n"
	createLogic += "\t\t// Handle error appropriately, e.g., log the error\n"
	createLogic += "\t\treturn\n"
	createLogic += "\t}\n\n"
	createLogic += "\tw.WriteHeader(http.StatusCreated)\n"
	createLogic += "}\n"

	return createLogic
}

func generateHTTPReadHandler(mainStruct StructInfo) string {
	readLogic := "func Read" + mainStruct.Name + "(w http.ResponseWriter, r *http.Request) {\n"
	readLogic += "\tif r.Method != http.MethodGet {\n"
	readLogic += "\t\tw.WriteHeader(http.StatusMethodNotAllowed)\n"
	readLogic += "\t\treturn\n"
	readLogic += "\t}\n\n"
	readLogic += "\tidFieldName := \"" + strings.ToLower(mainStruct.IDField) + "\"\n"
	readLogic += "\tid := r.URL.Query().Get(idFieldName) // extract ID from request parameters\n"
	readLogic += "\tif id == \"\" {\n"
	readLogic += "\t\tw.WriteHeader(http.StatusBadRequest)\n"
	readLogic += "\t\treturn\n"
	readLogic += "\t}\n\n"
	readLogic += "\trecord, exists := dataMap[id]\n"
	readLogic += "\tif !exists {\n"
	readLogic += "\t\tw.WriteHeader(http.StatusNotFound)\n"
	readLogic += "\t\treturn\n"
	readLogic += "\t}\n\n"
	readLogic += "\tjsonData, err := json.Marshal(record)\n"
	readLogic += "\tif err != nil {\n"
	readLogic += "\t\tw.WriteHeader(http.StatusInternalServerError)\n"
	readLogic += "\t\treturn\n"
	readLogic += "\t}\n\n"
	readLogic += "\tw.Header().Set(\"Content-Type\", \"application/json\")\n"
	readLogic += "\tw.WriteHeader(http.StatusOK)\n"
	readLogic += "\tw.Write(jsonData)\n"
	readLogic += "}\n"

	return readLogic
}

func generateHTTPUpdateHandler(mainStruct StructInfo) string {
	updateLogic := "func Update" + mainStruct.Name + "(w http.ResponseWriter, r *http.Request) {\n"
	updateLogic += "\tif r.Method != http.MethodPut {\n"
	updateLogic += "\t\tw.WriteHeader(http.StatusMethodNotAllowed)\n"
	updateLogic += "\t\treturn\n"
	updateLogic += "\t}\n\n"
	updateLogic += "\tvar updatedData " + mainStruct.Name + "\n"
	updateLogic += "\terr := json.NewDecoder(r.Body).Decode(&updatedData)\n"
	updateLogic += "\tif err != nil {\n"
	updateLogic += "\t\tw.WriteHeader(http.StatusBadRequest)\n"
	updateLogic += "\t\treturn\n"
	updateLogic += "\t}\n\n"
	updateLogic += "\t// Code to update the record in the dataMap\n"
	updateLogic += "\t// dataMap[updatedData." + mainStruct.IDField + "] = updatedData\n"
	updateLogic += "\terr = saveToJSONFile(\"" + strings.ToLower(mainStruct.Name) + ".json\")\n\n"
	updateLogic += "\tif err != nil {\n"
	updateLogic += "\t\tw.WriteHeader(http.StatusInternalServerError)\n"
	updateLogic += "\t\t// Handle error appropriately, e.g., log the error\n"
	updateLogic += "\t\treturn\n"
	updateLogic += "\tw.WriteHeader(http.StatusOK)\n"
	updateLogic += "}\n"

	return updateLogic
}

func generateHTTPDeleteHandler(mainStruct StructInfo) string {
	deleteLogic := "func Delete" + mainStruct.Name + "(w http.ResponseWriter, r *http.Request) {\n"
	deleteLogic += "\tif r.Method != http.MethodDelete {\n"
	deleteLogic += "\t\tw.WriteHeader(http.StatusMethodNotAllowed)\n"
	deleteLogic += "\t\treturn\n"
	deleteLogic += "\t}\n\n"
	deleteLogic += "\tidFieldName := \"" + strings.ToLower(mainStruct.IDField) + "\"\n"
	deleteLogic += "\tid := r.URL.Query().Get(idFieldName) // extract ID from request parameters\n"
	deleteLogic += "\tif id == \"\" {\n"
	deleteLogic += "\t\tw.WriteHeader(http.StatusBadRequest)\n"
	deleteLogic += "\t\treturn\n"
	deleteLogic += "\t}\n\n"
	deleteLogic += "\t// Code to delete the record from the dataMap using the extracted 'id'\n"
	deleteLogic += "\t// delete(dataMap, id)\n"
	deleteLogic += "\terr := saveToJSONFile(\"" + strings.ToLower(mainStruct.Name) + ".json\")\n\n"
	deleteLogic += "\tif err != nil {\n"
	deleteLogic += "\t\tw.WriteHeader(http.StatusInternalServerError)\n"
	deleteLogic += "\t\t// Handle error appropriately, e.g., log the error\n"
	deleteLogic += "\t\treturn\n"
	deleteLogic += "\t}\n\n"
	deleteLogic += "\tw.WriteHeader(http.StatusNoContent)\n"
	deleteLogic += "}\n"

	return deleteLogic
}

func main() {
	// Example input string containing package and struct definitions with spaces and newlines
	structsString := `
		package main
		
		type YourStruct struct {
			ID int
			Name string
			Email string
		}
		
		type AnotherStruct struct {
			Count int
			Value string
		}
	`

	jsonStructs, err := generateJSONNotation(structsString)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(jsonStructs)
}
