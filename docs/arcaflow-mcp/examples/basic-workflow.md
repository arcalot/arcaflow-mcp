# Tutorial: Conversational Input Construction

**Learn how to build valid workflow inputs through natural conversation**

This tutorial walks you through the complete workflow of discovering an Arcaflow workflow, understanding its schema, and building valid inputs conversationally with AI assistance.

**Duration:** 10-15 minutes  
**Prerequisites:** Arcaflow MCP server installed and configured  
**What you'll learn:**
- How to load and inspect workflows
- How to understand workflow schemas  
- How to build inputs conversationally
- How to validate and export inputs

---

## Overview

Building workflow inputs manually requires:
1. Understanding complex YAML/JSON schemas
2. Knowing valid value ranges and types
3. Ensuring all required fields are present
4. Validating against schema constraints

With Arcaflow MCP, you can instead:
1. **Describe what you want to test** in natural language
2. **Let the AI guide you** through required fields
3. **Get instant validation** as you build
4. **Export ready-to-use** YAML/JSON

---

## Step 1: Load a Workflow

First, we need to tell the MCP server which workflow we want to work with. Workflows can be loaded from:
- Local filesystem
- Git repositories
- HTTP URLs

### Using the Built-In Example Workflows

The Arcaflow MCP repository includes example workflows in `examples/workflows/`:
- **hello-world/** - Simple single-input workflow (perfect for this tutorial)
- **data-processing/** - More complex workflow with multiple inputs
- **perf-test/** - Advanced workflow for performance testing

### Example Conversation:

**You:**
> Load the hello-world workflow from /path/to/arcaflow-mcp/examples/workflows/hello-world

**Replace `/path/to/arcaflow-mcp`** with your actual installation path, such as:
- `/home/user/arcaflow-mcp` (Linux)
- `/Users/user/arcaflow-mcp` (macOS)
- `C:\Users\user\arcaflow-mcp` (Windows)

**AI (using `workflow_load` tool):**
> I've loaded the hello-world workflow. This is a simple workflow that:
> - Accepts a `name` parameter (string)
> - Generates a greeting message
> - Returns the greeting as output
>
> Would you like me to show you the input schema?

### What Happened?

The MCP server:
1. Located the workflow file (`workflow.yaml`)
2. Parsed the workflow definition
3. Extracted the input schema
4. Made it available for conversation

---

## Step 2: Understand the Schema

Before building inputs, it helps to understand what the workflow expects.

### Example Conversation:

**You:**
> What inputs does this workflow require?

**AI (using `workflow_schema_get` tool):**
> The hello-world workflow requires these inputs:
>
> **Required Fields:**
> - `name` (string) - The name to greet
>   - Must be a non-empty string
>   - Description: "Name to greet"
>
> **Optional Fields:**
> - None
>
> This is a simple workflow with just one required field.

### What Happened?

The MCP server:
1. Retrieved the workflow's input schema
2. Identified required vs optional fields
3. Extracted field types and constraints
4. Provided human-readable descriptions

---

## Step 3: Build Inputs Conversationally

Now you can build the inputs by describing what you want, and the AI will help construct valid input data.

### Example Conversation:

**You:**
> I want to test the workflow with the name "Alice"

**AI (using `workflow_input_build` tool):**
> I've created workflow inputs with name set to "Alice". 
>
> The current input:
>
> ```yaml
> name: "Alice"
> ```
>
> This input is valid and ready to use. Would you like to:
> - Export it to a file?
> - Validate it one more time?
> - Modify any values?

### Alternative: Iterative Building

For more complex workflows, you can build inputs iteratively:

**You:**
> Start building inputs for the workflow

**AI:**
> Let's build the inputs step by step.
>
> First, I need a value for the required field `name` (string).
> What name would you like to use?

**You:**
> Use "Bob"

**AI (using `workflow_input_build`):**
> Added name: "Bob"
>
> All required fields are now filled. The input is:
>
> ```yaml
> name: "Bob"
> ```
>
> Ready to export or validate?

### What Happened?

The MCP server:
1. Validated each value against the schema
2. Built a structured input object
3. Maintained state across the conversation
4. Provided immediate feedback on validity

---

## Step 4: Validate Inputs

Before exporting, you can explicitly validate the inputs to ensure they'll work with Arcaflow Engine.

### Example Conversation:

**You:**
> Validate the inputs

**AI (using `workflow_input_validate` tool):**
> ✓ Validation successful!
>
> The inputs are valid and meet all schema requirements:
> - All required fields are present
> - All field types are correct
> - All constraints are satisfied
>
> You can safely export these inputs for use with Arcaflow Engine.

### What if Validation Fails?

**AI:**
> ✗ Validation failed:
>
> Error: Field `name` is required but not provided
>
> Would you like me to help you add the missing field?

### What Happened?

The MCP server:
1. Performed schema validation
2. Checked type constraints
3. Verified required fields
4. Reported any issues clearly

---

## Step 5: Export Inputs

Once inputs are validated, export them to a file for use with Arcaflow Engine.

### Example Conversation:

**You:**
> Export the inputs to hello-inputs.yaml

**AI (using `workflow_input_export` tool):**
> ✓ Exported inputs to hello-inputs.yaml
>
> The file contains:
>
> ```yaml
> name: "Alice"
> ```
>
> You can now run the workflow with:
>
> ```bash
> arcaflow -input hello-inputs.yaml -workflow /path/to/workflow.yaml
> ```

### Export Options

You can export in different formats:

**YAML (default):**
> Export inputs to hello-inputs.yaml

**JSON:**
> Export inputs as JSON to hello-inputs.json

### What Happened?

The MCP server:
1. Serialized the input data
2. Wrote it to the specified file
3. Used the requested format (YAML/JSON)
4. Confirmed the export

---

## Complete Example Flow

Here's a complete conversation showing the entire process:

**You:**
> Load the hello-world workflow from /path/to/examples/workflows/hello-world,
> build inputs with name "Alice", and export to hello-inputs.yaml

**AI:**
> I'll help you with that. Let me:
>
> 1. Load the workflow...
>    ✓ Workflow loaded: hello-world
>
> 2. Build inputs with name="Alice"...
>    ✓ Input constructed and validated
>
> 3. Export to hello-inputs.yaml...
>    ✓ File written
>
> Your workflow inputs are ready! The file contains:
>
> ```yaml
> name: "Alice"
> ```
>
> Run the workflow with:
>
> ```bash
> arcaflow -input hello-inputs.yaml -workflow /path/to/workflow.yaml
> ```

---

## Next Steps

Now that you know how to build inputs conversationally, try:

**More Complex Workflows:**
- Try the [data-processing workflow](https://github.com/arcalot/arcaflow-mcp/tree/main/examples/workflows/data-processing) with multiple fields
- Explore workflows with nested objects and arrays

**Iterative Optimization:**
- Run your workflow with the generated inputs
- Analyze the results (see [Tutorial: Iterative Optimization](iterative-optimization.md))
- Refine inputs based on AI suggestions

**Team Workflows:**
- Deploy in [Server Mode](../usage/server-mode.md) for team access
- Share workflows and inputs across your team

---

## Tips and Best Practices

### Working with Complex Schemas

For workflows with many fields:
1. Start with required fields
2. Add optional fields as needed
3. Use `workflow_schema_get` to understand structure
4. Build iteratively, validating along the way

### Schema Exploration

To understand a workflow before building:

> Show me the complete input schema for this workflow

The AI will provide:
- All fields (required and optional)
- Data types and constraints
- Nested object structures
- Array specifications
- Default values

### Modifying Existing Inputs

You can modify inputs after building:

> Change the name to "Carol"

The AI will:
1. Update the field
2. Revalidate
3. Show the updated inputs

### Troubleshooting

**"Workflow not found"**
- Check the path is correct
- Ensure `workflow.yaml` exists in the directory
- Use absolute paths when possible

**"Invalid input value"**
- Check the schema for allowed values/types
- Ask the AI: "What are valid values for field X?"
- Review constraint error messages

**"Session lost"**
- If using server mode without SSE, state isn't maintained between requests
- For multi-step workflows, use a persistent connection or local mode

---

## Related Documentation

- **[Input Construction Guide](../usage/input-construction.md)** - Detailed guide to all input construction features
- **[Tool Reference: Input Tools](../tools/input-tools.md)** - Complete reference for all input-related MCP tools
- **[Workflow Examples](https://github.com/arcalot/arcaflow-mcp/tree/main/examples/workflows)** - More example workflows to try
- **[Result Analysis Tutorial](iterative-optimization.md)** - Learn to analyze workflow results

---

**Questions or issues?** Check the [FAQ](../faq.md) or [Troubleshooting Guide](../troubleshooting.md).
