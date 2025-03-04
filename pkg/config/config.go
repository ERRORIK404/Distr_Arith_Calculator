package config

import (
	"os"
	"strconv"

	env "github.com/joho/godotenv"
)

type Config struct {
	TIME_ADDITION_MS int  // Время выполнения слоежния в миллисекундах
	TIME_SUBTRACTION_MS int // Время выполнения вычитание в миллисекундах
	TIME_MULTIPLICATIONS_MS int // Время выполения умножения в миллисекундах
	TIME_DIVISIONS_MS int // Время выполнения деления в миллисекундах
	COMPUTING_POWER int // Количество агентов, то есть количество серверов, которые могут обрабатывать конкретные задачи
}


func LoadConfig() *Config {
	err := env.Load("environment_variables.env")
    if err != nil {
        panic("Error loading .env file")
    }
	
	add, err := strconv.Atoi(os.Getenv("TIME_ADDITION_MS"))
	if err != nil {
        panic("Invalid TIME_ADDITION_MS")
    }

	sub, err := strconv.Atoi(os.Getenv("TIME_SUBTRACTION_MS"))
	if err != nil {
        panic("Invalid TIME_SUBTRACTION_MS")
    }

	mul, err := strconv.Atoi(os.Getenv("TIME_MULTIPLICATIONS_MS"))
	if err != nil {
        panic("Invalid TIME_MULTIPLICATIONS_MS")
    }

	div, err := strconv.Atoi(os.Getenv("TIME_DIVISIONS_MS"))
	if err != nil {
        panic("Invalid TIME_DIVISIONS_MS")
    }

	pow, err := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	if err != nil {
        panic("Invalid COMPUTING_POWER")
    }

	return &Config{
		TIME_ADDITION_MS: add,
        TIME_SUBTRACTION_MS: sub,
        TIME_MULTIPLICATIONS_MS: mul,
        TIME_DIVISIONS_MS: div,
        COMPUTING_POWER: pow,
	}
}