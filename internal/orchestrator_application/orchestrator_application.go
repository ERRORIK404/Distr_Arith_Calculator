package orchestratorapplication

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	agent "github.com/ERRORIK404/Distr_Arith_Calculator/internal/agent_application"
	conf "github.com/ERRORIK404/Distr_Arith_Calculator/pkg/config"
	conv "github.com/ERRORIK404/Distr_Arith_Calculator/pkg/converter_to_RPN"
	locerr "github.com/ERRORIK404/Distr_Arith_Calculator/pkg/local_errors"
	structs "github.com/ERRORIK404/Distr_Arith_Calculator/pkg/structs"
)

var (
	config = conf.LoadConfig()
	tasksMap = structs.SafeTaskMap{Task_map: make(map[int]structs.Task)}
	expressionsMap = structs.SafeExpressionMap{Expression_map: make(map[int]structs.Expression)}
)



// Структура дерева нужна чтобы конвертировать RPN в форму удобную для подсчета выражения с использованием concurrency кода
type ExprNode struct {
	Left  *ExprNode
	Right *ExprNode
	Op    string
	Value float64
}

// Конвертируем RPN запись в дерево для того, чтобы можно было асинхронно подсчитывать ветви
func RpnToTree(tokens []string) (*ExprNode, error) {
	stack := []*ExprNode{}

	for _, token := range tokens {
		if num, err := strconv.ParseFloat(token, 64); err == nil {
			// Если токен — число, создаем лист и добавляем в стек
			stack = append(stack, &ExprNode{Value: num})
		} else {
			// Если токен — оператор, извлекаем два операнда из стека
			if len(stack) < 2 {
				return nil, fmt.Errorf("недостаточно операндов для оператора %s", token)
			}

			// Извлекаем правый и левый операнды
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			// Создаем новый узел и добавляем его в стек
			node := &ExprNode{Op: token, Left: left, Right: right}
			stack = append(stack, node)
		}
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("некорректное выражение")
	}

	return stack[0], nil
}

// Функция для асинхронного подсчета ветвей
func (n *ExprNode) EvaluateParallel() float64 {
	if n.Left == nil && n.Right == nil {
		// Еслт это лист, возвращаем значение
		return n.Value
	}

	// Каналы для получения результатов
	leftChan := make(chan float64, 15)
	rightChan := make(chan float64, 15)

	// Запускаем вычисление левого поддерева
	go func() {
		leftChan <- n.Left.EvaluateParallel()
	}()

	// Запускаем вычисление правого поддерева
	go func() {
		rightChan <- n.Right.EvaluateParallel()
	}()

	// Ждем результаты
	left := <-leftChan
	right := <-rightChan

	// Создаем задачу, и ждем исполнения
	log.Println("CREATE TASK")
	switch n.Op {
		case "+":
			task := structs.NewTask(left, right, "+", config.TIME_ADDITION_MS)
			tasksMap.Write(task)
			return <- tasksMap.Read(task.Id).Chan
		case "-":
			task := structs.NewTask(left, right, "-", config.TIME_SUBTRACTION_MS)
			tasksMap.Write(task)
			return <- tasksMap.Read(task.Id).Chan
		case "*":
			task := structs.NewTask(left, right, "*", config.TIME_MULTIPLICATIONS_MS)
			tasksMap.Write(task)
			return <- tasksMap.Read(task.Id).Chan
		case "/":
			if right == 0 {
				return left
			}
			task := structs.NewTask(left, right, "/", config.TIME_DIVISIONS_MS)
			tasksMap.Write(task)
			return <- tasksMap.Read(task.Id).Chan
		default:
			return 0
	}
}


// Функция для вылидации и полного подсчета выражения
func Calc(expression string) (res float64, err error) {
    // Удаляем пробелы из выражения
    expression = strings.ReplaceAll(expression, " ", "")

    // Проверка на пустоту
    if expression == "" {
        return 0, locerr.ErrEmptyExpression
    }

    // Проверка на правильность скобок
    if !conv.IsValidParentheses(expression) {
        return 0,locerr.ErrIncorrectBracketPlacement
    }

    // Преобразование выражения в RPN
    rpn, err := conv.InfixToRPN(expression)
    if err != nil {
        return 0, err
    }

    // Преобразование обратной польской нотации в дерево
    result, err := RpnToTree(rpn)
    if err != nil {
        return 0, err
    }
	// Вычисляем дерево
    return result.EvaluateParallel(), nil
}


// Хендлер для получчения нового выражения
func Accept_expression_handler(w http.ResponseWriter, r *http.Request) {
	expression := structs.NewExpression()
	type message struct{Message string `json:"expression"`}
	var str message

    defer r.Body.Close()
    err := json.NewDecoder(r.Body).Decode(&str)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnprocessableEntity)
        return
    }

	expression.Expression = str.Message
	expressionsMap.Write(expression)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	type id struct{Id int `json:"id"`}
	json.NewEncoder(w).Encode(id{Id:expression.Id})

	// Запускаем вычисление выражения в отдельном потоке
    go func(){
        result, err := Calc(expression.Expression)
        if err != nil {
			expression.Status = "error"
			if err == locerr.ErrIncorrectBracketPlacement || err == locerr.ErrEmptyExpression{
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
            return
        }
		expression.Result = fmt.Sprintf("%f", result)
		expression.Status = "success"
		expressionsMap.Write(expression)
    }()
}


// Хендер для получения агентом выражения и приема результата от агента в зависимости от http метода
func Assigned_task_handler(w http.ResponseWriter, r *http.Request){
	if r.Method == "GET" {
		task, err := tasksMap.Get_does_not_have_result()
		if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }
		json.NewEncoder(w).Encode(task)
    } else if r.Method == "POST"{
		var res structs.Result
		json.NewDecoder(r.Body).Decode(&res)	
		tasksMap.Write_result(res)
		log.Println(tasksMap.Read(res.Id))
	}
}

// Хендлер для получения истории всех выражений
func Get_all_expressions_handler(w http.ResponseWriter, r *http.Request) {
    expressionsMap.ExpressionMutex.RLock()
    defer expressionsMap.ExpressionMutex.RUnlock()

    expressions := make([]structs.Expression, 0, len(expressionsMap.Expression_map))
    for _, expression := range expressionsMap.Expression_map {
        expressions = append(expressions, expression)
    }

    jsonData, err := json.Marshal(expressions)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(jsonData)
	log.Println("Get all data from map")
	log.Println(tasksMap)
}

//Хендре для получения выражения по его индентификатору
func Get_expression_by_id_handler(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(strings.Split(r.URL.Path, "/")[4][1:])
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    expression, err := expressionsMap.Read(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    jsonData, err := json.Marshal(expression)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(jsonData)
}


func RunServer(){
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/expressions", Get_all_expressions_handler)
	mux.HandleFunc("/api/v1/calculate", Accept_expression_handler)
    mux.HandleFunc("/internal/task", Assigned_task_handler)
	mux.HandleFunc("/api/v1/expressions/", Get_expression_by_id_handler)
	go func(){http.ListenAndServe(":8080", mux)}()

	for i := 0; i < config.COMPUTING_POWER; i++ {
		go func() {
			if agent.AgentRun() != nil{
				fmt.Println("Agent kill")
			}
		}()
	}
	for {
		time.Sleep(time.Minute)
	}
}
