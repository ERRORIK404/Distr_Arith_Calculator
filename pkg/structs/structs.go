package structs

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
)

func GenerateID() int {
    var randomUint uint64
	binary.Read(rand.Reader, binary.BigEndian, &randomUint)
	id := int(randomUint)
	if id < 0 {
        id = -id
    }
	return id
}



// Структура для хранения арифметического выражения
type Expression struct {
	Id int `json:"id"`
	Expression string `json:"expression"`
	Status string `json:"status"`
	Result string `json:"result"`
}

func NewExpression() Expression {
	return Expression{Id: GenerateID(), Status: "Pending"}
}



// Структура для задач которые выполняет агент
type Task struct {
	Id int `json:"id"`
	Operand1 float64 `json:"arg1"`
	Operand2 float64 `json:"arg2"`
	Operation string `json:"operation"`
	Operation_time int `json:"operation_time"`
	Status string `json:"-"`
	Chan chan float64 `json:"-"`
}

func NewTask(operand1, operand2 float64, operation string, operation_time int) Task{
    id := GenerateID()
    task := Task{id, operand1, operand2, operation, operation_time,"Pending", make(chan float64, 1)}
	return task
}



// Структура для результата выполненной задачи
type Result struct{
	Id int `json:"id"`
	Result float64 `json:"result"`
}




// Структура для хранения всех выражений, которая безопасна для чтений и записи с использование конкуретного кода
type SafeExpressionMap struct {
	Expression_map map[int]Expression
	ExpressionMutex sync.RWMutex
}

func (m *SafeExpressionMap) Write(expression Expression){
	m.ExpressionMutex.Lock()
    defer m.ExpressionMutex.Unlock()
    m.Expression_map[expression.Id] = expression
}

func (m *SafeExpressionMap) Read(id int) (Expression, error){
	m.ExpressionMutex.RLock()
    defer m.ExpressionMutex.RUnlock()
	if _, ok := m.Expression_map[id];!ok {
        return Expression{}, errors.New("Expression not found")
    }
    return m.Expression_map[id], nil
}




// Структура для хранения всех задач, которая безопасна для чтений и записи с использование конкуретного кода
type SafeTaskMap struct {
	Task_map map[int]Task
	TaskMutex sync.RWMutex
}

func (m *SafeTaskMap) Write(task Task){
	m.TaskMutex.Lock()
    defer m.TaskMutex.Unlock()
    m.Task_map[task.Id] = task
}

func (m *SafeTaskMap) Read(id int) Task{
	m.TaskMutex.RLock()
    defer m.TaskMutex.RUnlock()
    return m.Task_map[id]
}

// Метод для получения ожидающей выполнения задачи
func (m *SafeTaskMap) Get_does_not_have_result() (Task, error){
	m.TaskMutex.Lock()
    defer m.TaskMutex.Unlock()
    for _, task := range m.Task_map {
        if task.Status == "Pending" {
			task.Status = "Progress"
			m.Task_map[task.Id] = task
            return task, nil
        }
    }
    return Task{}, errors.New("All tasks are in progress")
}

func (m *SafeTaskMap) Write_result(res Result){
	m.TaskMutex.Lock()
    defer m.TaskMutex.Unlock()
    task, ok := m.Task_map[res.Id]
    if ok {
        task.Status = "done"
        task.Chan <- res.Result
		m.Task_map[res.Id] = task
    }
}