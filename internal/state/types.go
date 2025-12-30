// Package state provides types and utilities for working with Terraform state files.
//
// This package handles the representation and parsing of Terraform state files
// (version 4 and later). It provides structured types that map to the JSON
// schema of Terraform state files, enabling type-safe access to state data.
//
// The primary types are:
//   - StateFile: Represents a complete Terraform state file
//   - OutputValue: Represents a single output value with its type and sensitivity
//
// State files are immutable once parsed, and all fields are exported for
// JSON marshaling/unmarshaling.
package state

// StateFile represents a Terraform state file (version 4 or later).
//
// This structure maps to the JSON schema of Terraform state files as documented at:
// https://www.terraform.io/internals/json-format#state-representation
//
// Only fields required for output value access are included. Additional fields
// (resources, modules, etc.) are available in the raw state data but not
// parsed by this provider.
//
// Example state file structure:
//
//	{
//	  "version": 4,
//	  "terraform_version": "1.5.0",
//	  "serial": 42,
//	  "lineage": "abc123-def456-...",
//	  "outputs": {
//	    "vpc_id": {
//	      "value": "vpc-12345",
//	      "type": "string",
//	      "sensitive": false
//	    }
//	  }
//	}
//
//nolint:revive // StateFile follows Terraform's naming convention
type StateFile struct {
	// Version is the state file format version. Must be 4 or greater.
	// Version 3 and earlier are not supported by this provider.
	Version int `json:"version"`

	// TerraformVersion is the version of Terraform that wrote this state file.
	// Format: "major.minor.patch" (e.g., "1.5.0")
	TerraformVersion string `json:"terraform_version"`

	// Serial is an incrementing number that changes with each state write.
	// Used by Terraform for conflict detection and state locking.
	Serial int `json:"serial"`

	// Lineage is a UUID that uniquely identifies this state file's history.
	// All state files derived from the same initial state share the same lineage.
	Lineage string `json:"lineage"`

	// Outputs contains all output values defined in the root module.
	// Key is the output name, value is the OutputValue structure.
	// Outputs from child modules are not included in this map.
	Outputs map[string]*OutputValue `json:"outputs"`
}

// OutputValue represents a single Terraform output value.
//
// Outputs are values explicitly exported by a Terraform configuration
// using the "output" block. Each output has a value (of any JSON type),
// a type annotation, and a sensitivity flag.
//
// The Type field uses Terraform's type system notation:
//   - Primitives: "string", "number", "bool"
//   - Collections: "list(string)", "map(number)", "set(bool)"
//   - Structural: "object({...})", "tuple([...])"
//
// Example output value:
//
//	{
//	  "value": "vpc-12345",
//	  "type": "string",
//	  "sensitive": false
//	}
type OutputValue struct {
	// Value is the actual output value. Can be any JSON-compatible type:
	// string, number, bool, array, object, or null.
	//
	// The Go type will be:
	//   - string for Terraform strings
	//   - float64 for Terraform numbers
	//   - bool for Terraform bools
	//   - []interface{} for Terraform lists/tuples/sets
	//   - map[string]interface{} for Terraform maps/objects
	//   - nil for Terraform null
	Value interface{} `json:"value"`

	// Type is the Terraform type annotation for this value.
	// Uses Terraform's type system notation (e.g., "string", "list(string)", "map(number)").
	//
	// This is informational and not enforced by this provider. The actual
	// Value field contains the JSON representation which may not perfectly
	// preserve Terraform's type distinctions (e.g., lists vs. tuples).
	Type string `json:"type"`

	// Sensitive indicates whether this output is marked as sensitive.
	// Sensitive outputs should be handled carefully and not logged or
	// exposed in plain text.
	//
	// Note: The value itself is still present in this structure. It's
	// the responsibility of the caller to handle sensitive values appropriately.
	Sensitive bool `json:"sensitive"`
}
