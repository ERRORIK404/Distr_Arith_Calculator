package agentapplication

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	structs "github.com/ERRORIK404/Distr_Arith_Calculator/pkg/structs"
)
type Agent struct {
  Agent http.Client
}

func New() *Agent {
  return &Agent{http.Client{}}
}

func AgentRun() error{
	url := "http://localhost:8080/internal/task"
	ch := make(chan struct{})
	go func() {
		for {
			a := New()
			time.Sleep(time.Second)
			response := new(http.Response)
			response, err := a.Agent.Get(url)
            if response.StatusCode != http.StatusOK ||  err != nil{
                log.Println("Unexpected status code:", response.StatusCode)
                continue
            }
			var task structs.Task
			err = json.NewDecoder(response.Body).Decode(&task)
			response.Body.Close()
			timer := time.NewTimer(time.Duration(task.Operation_time) * time.Millisecond)	
			var result structs.Result
			result.Id = task.Id
			switch task.Operation {
				case "+":
					result.Result = task.Operand1 + task.Operand2
				case "-":
					result.Result = task.Operand1 - task.Operand2
				case "*":
					result.Result = task.Operand1 * task.Operand2
				case "/":
					result.Result = task.Operand1 / task.Operand2
			}
			jsonData, _ := json.Marshal(result)
			<- timer.C
			http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		}
	}()
	<-ch
	defer log.Println("DIED")
	return errors.New("agent died")
}