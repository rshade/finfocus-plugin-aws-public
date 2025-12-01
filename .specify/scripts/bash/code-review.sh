#!/usr/bin/env bash

# Code Review Command for opencode
# Provides automated code review feedback based on best practices

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
TARGET="."
FORMAT="text"
SEVERITY="medium"
LANGUAGE=""
FIX=false

# Help function
show_help() {
    cat << EOF
Usage: /code-review [OPTIONS] [TARGET]

Perform automated code review on specified target.

OPTIONS:
    -h, --help          Show this help message
    -f, --format        Output format: text, json, markdown (default: text)
    -s, --severity      Minimum severity level: low, medium, high (default: medium)
    -l, --language      Language override (auto-detected by default)
    --fix               Suggest fixes for issues when possible

TARGET:
    File or directory to review (default: current directory)

EXAMPLES:
    /code-review
    /code-review src/
    /code-review --severity high src/main.go
    /code-review --format json --fix src/

EXIT CODES:
    0  Success - review completed
    1  Error - review failed or critical issues found
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -f|--format)
            FORMAT="$2"
            shift 2
            ;;
        -s|--severity)
            SEVERITY="$2"
            shift 2
            ;;
        -l|--language)
            LANGUAGE="$2"
            shift 2
            ;;
        --fix)
            FIX=true
            shift
            ;;
        -*)
            echo "Unknown option $1"
            show_help
            exit 1
            ;;
        *)
            TARGET="$1"
            shift
            ;;
    esac
done

# Detect language if not specified
detect_language() {
    local target="$1"
    if [[ -n "$LANGUAGE" ]]; then
        echo "$LANGUAGE"
        return
    fi
    
    if [[ -d "$target" ]]; then
        if find "$target" -name "*.go" -type f | head -1 | grep -q .; then
            echo "go"
        elif find "$target" -name "*.js" -o -name "*.ts" -type f | head -1 | grep -q .; then
            echo "javascript"
        elif find "$target" -name "*.py" -type f | head -1 | grep -q .; then
            echo "python"
        else
            echo "unknown"
        fi
    elif [[ -f "$target" ]]; then
        case "$target" in
            *.go) echo "go" ;;
            *.js|*.ts) echo "javascript" ;;
            *.py) echo "python" ;;
            *) echo "unknown" ;;
        esac
    else
        echo "unknown"
    fi
}

# Go-specific review
review_go() {
    local target="$1"
    local severity="$2"
    local fix="$3"
    
    echo -e "${BLUE}Performing Go code review on: $target${NC}"
    echo "----------------------------------------"
    
    # Check for common Go issues
    local issues=0
    
    # Check for error handling
    if [[ "$severity" == "low" ]] || [[ "$severity" == "medium" ]]; then
        if grep -r "^\s*if.*!= nil" "$target" > /dev/null 2>&1; then
            echo -e "${YELLOW}MEDIUM: Consider adding error handling comments for clarity${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Add comments explaining what happens in each error case"
            fi
            ((issues++))
        fi
    fi
    
    # Check for proper error checking
    if grep -r "^\s*err :=" "$target" > /dev/null 2>&1; then
        if ! grep -r "^\s*if err != nil" "$target" > /dev/null 2>&1; then
            echo -e "${RED}HIGH: Found error assignment without error checking${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Add error checking immediately after error assignment:"
                echo -e "    if err != nil {"
                echo -e "        // Handle error appropriately"
                echo -e "        return err // or handle as appropriate"
                echo -e "    }"
            fi
            ((issues++))
        fi
    fi
    
    # Check for proper context usage
    if grep -r "context\." "$target" > /dev/null 2>&1; then
        if ! grep -r "context\..*Cancel" "$target" > /dev/null 2>&1; then
            echo -e "${YELLOW}MEDIUM: Context usage detected, ensure proper cancellation${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Use context.WithCancel or context.WithTimeout and call cancel()"
                echo -e "    ctx, cancel := context.WithCancel(context.Background())"
                echo -e "    defer cancel()"
            fi
            ((issues++))
        fi
    fi
    
    # Check for proper mutex usage
    if grep -r "sync\." "$target" > /dev/null 2>&1; then
        if grep -r "Mutex" "$target" > /dev/null 2>&1; then
            if ! grep -r "Lock()" "$target" > /dev/null 2>&1 || ! grep -r "Unlock()" "$target" > /dev/null 2>&1; then
                echo -e "${RED}HIGH: Mutex declared but Lock/Unlock not found${NC}"
                if [[ "$fix" == "true" ]]; then
                    echo -e "  Suggestion: Ensure every Lock() has a corresponding Unlock():"
                    echo -e "    mu.Lock()"
                    echo -e "    // critical section"
                    echo -e "    mu.Unlock()"
                fi
                ((issues++))
            fi
        fi
    fi
    
    # Check for proper defer usage with file operations
    if grep -r "os\..*Open\|ioutil\." "$target" > /dev/null 2>&1; then
        if ! grep -r "defer.*Close" "$target" > /dev/null 2>&1; then
            echo -e "${YELLOW}MEDIUM: File operations detected, consider using defer for Close${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Use defer to ensure file is closed:"
                echo -e "    file, err := os.Open(\"filename\")"
                echo -e "    if err != nil {"
                echo -e "        return err"
                echo -e "    }"
                echo -e "    defer file.Close()"
            fi
            ((issues++))
        fi
    fi
    
    echo "----------------------------------------"
    echo -e "${GREEN}Go code review completed with $issues potential issues${NC}"
}

# JavaScript/TypeScript review
review_javascript() {
    local target="$1"
    local severity="$2"
    local fix="$3"
    
    echo -e "${BLUE}Performing JavaScript/TypeScript code review on: $target${NC}"
    echo "----------------------------------------"
    
    local issues=0
    
    # Check for proper async/await usage
    if grep -r "async.*function\|Promise\." "$target" > /dev/null 2>&1; then
        if grep -r "\.then\|\.catch" "$target" > /dev/null 2>&1; then
            echo -e "${YELLOW}MEDIUM: Mixed Promise and async/await usage detected${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Use consistent async/await pattern:"
                echo -e "    // Instead of promise.then()"
                echo -e "    // Use async/await:"
                echo -e "    try {"
                echo -e "        const result = await someAsyncFunction();"
                echo -e "    } catch (error) {"
                echo -e "        // Handle error"
                echo -e "    }"
            fi
            ((issues++))
        fi
    fi
    
    # Check for proper error handling
    if grep -r "try.*{.*}.*catch" "$target" > /dev/null 2>&1; then
        if ! grep -r "catch.*(" "$target" > /dev/null 2>&1; then
            echo -e "${YELLOW}LOW: Consider naming your catch variables for better error handling${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Name your catch variables:"
                echo -e "    try {"
                echo -e "        // some code"
                echo -e "    } catch (error) {"
                echo -e "        console.error('Error:', error.message);"
                echo -e "    }"
            fi
            ((issues++))
        fi
    fi
    
    # Check for var usage
    if grep -r "var " "$target" > /dev/null 2>&1; then
        echo -e "${YELLOW}LOW: Consider using let/const instead of var${NC}"
if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Replace 'var' with 'let' (for reassignable variables) or 'const' (for constants)"
            fi
        ((issues++))
    fi
    
    echo "----------------------------------------"
    echo -e "${GREEN}JavaScript/TypeScript code review completed with $issues potential issues${NC}"
}

# Python review
review_python() {
    local target="$1"
    local severity="$2"
    local fix="$3"
    
    echo -e "${BLUE}Performing Python code review on: $target${NC}"
    echo "----------------------------------------"
    
    local issues=0
    
    # Check for proper exception handling
    if grep -r "try:" "$target" > /dev/null 2>&1; then
        if ! grep -r "except.*:" "$target" > /dev/null 2>&1; then
            echo -e "${YELLOW}MEDIUM: Try block found without except handler${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Add appropriate exception handling:"
                echo -e "    try:"
                echo -e "        # some code"
                echo -e "    except SpecificException as e:"
                echo -e "        # handle specific exception"
                echo -e "    except Exception as e:"
                echo -e "        # handle general exception"
            fi
            ((issues++))
        fi
    fi
    
    # Check for proper context manager usage
    if grep -r "open(" "$target" > /dev/null 2>&1; then
        if ! grep -r "with.*open" "$target" > /dev/null 2>&1; then
            echo -e "${YELLOW}MEDIUM: File operations detected, consider using 'with' statement${NC}"
            if [[ "$fix" == "true" ]]; then
                echo -e "  Suggestion: Use 'with' statement for automatic file closing:"
                echo -e "    # Instead of:"
                echo -e "    # file = open('filename', 'r')"
                echo -e "    # content = file.read()"
                echo -e "    # file.close()"
                echo -e "    # Use:"
                echo -e "    with open('filename', 'r') as file:"
                echo -e "        content = file.read()"
            fi
            ((issues++))
        fi
    fi
    
    echo "----------------------------------------"
    echo -e "${GREEN}Python code review completed with $issues potential issues${NC}"
}

# Main execution
main() {
    local lang
    lang=$(detect_language "$TARGET")
    
    echo -e "${BLUE}Code Review Tool${NC}"
    echo "Target: $TARGET"
    echo "Language: $lang"
    echo "Severity: $SEVERITY"
    echo "Format: $FORMAT"
    echo "Fix suggestions: $FIX"
    echo ""
    
    case "$lang" in
        "go")
            review_go "$TARGET" "$SEVERITY" "$FIX"
            ;;
        "javascript")
            review_javascript "$TARGET" "$SEVERITY" "$FIX"
            ;;
        "python")
            review_python "$TARGET" "$SEVERITY" "$FIX"
            ;;
        *)
            echo -e "${YELLOW}Unsupported language or no code files found${NC}"
            echo "Supported languages: Go, JavaScript/TypeScript, Python"
            exit 1
            ;;
    esac
}

# Run main function
main
