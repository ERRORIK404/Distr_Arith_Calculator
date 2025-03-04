package convertertoRpn

import (
	"strings"

	locerr "github.com/ERRORIK404/Distr_Arith_Calculator/pkg/local_errors"
)

// Функция для проверки правильности расставления скобок
func IsValidParentheses(expression string) bool {
    stack := []rune{}
    for _, char := range expression {
        switch char {
        case '(':
            stack = append(stack, char)
        case ')':
            if len(stack) == 0 || stack[len(stack)-1] != '(' {
                return false
            }
            stack = stack[:len(stack)-1]
        }
    }
    return len(stack) == 0
}


// Функция для преобразования выражения в обратную польскую нотацию
func InfixToRPN(expression string) ([]string, error) {
    rpn := []string{}
    operatorStack := []string{}
    numberBuffer := strings.Builder{}

    for _, char := range expression {
        switch {
        case IsDigit(char) || char == '.':
            numberBuffer.WriteRune(char)
        case char == '+' || char == '-' || char == '*' || char == '/':
            if numberBuffer.Len() > 0 {
                rpn = append(rpn, numberBuffer.String())
                numberBuffer.Reset()
            }
            for len(operatorStack) > 0 && GetPrecedence(string(char)) <= GetPrecedence(operatorStack[len(operatorStack)-1]) {
                rpn = append(rpn, operatorStack[len(operatorStack)-1])
                operatorStack = operatorStack[:len(operatorStack)-1]
            }
            operatorStack = append(operatorStack, string(char))
        case char == '(':
            operatorStack = append(operatorStack, string(char))
        case char == ')':
            if numberBuffer.Len() > 0 {
                rpn = append(rpn, numberBuffer.String())
                numberBuffer.Reset()
            }
            for len(operatorStack) > 0 && operatorStack[len(operatorStack)-1] != "(" {
                rpn = append(rpn, operatorStack[len(operatorStack)-1])
                operatorStack = operatorStack[:len(operatorStack)-1]
            }
            if len(operatorStack) == 0 {
                return nil, locerr.ErrBracketMismatch
            }
            operatorStack = operatorStack[:len(operatorStack)-1]
        default:
            return nil, locerr.ErrInvalidCharacter
        }
    }

    if numberBuffer.Len() > 0 {
        rpn = append(rpn, numberBuffer.String())
    }

    for len(operatorStack) > 0 {
        rpn = append(rpn, operatorStack[len(operatorStack)-1])
        operatorStack = operatorStack[:len(operatorStack)-1]
    }

    return rpn, nil
}

func IsDigit(char rune) bool {
    return char >= '0' && char <= '9'
}

func GetPrecedence(operator string) int {
    switch operator {
    case "+", "-":
        return 1
    case "*", "/":
        return 2
    default:
        return 0
    }
}
