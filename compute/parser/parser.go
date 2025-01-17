package parser

import (
	"errors"
	"strings"
	"unicode"
)

// CommandType – перечисление возможных типов команд
type CommandType int

const (
	SET CommandType = iota
	GET
	DEL
)

// Command – структура, описывающая распарсенную команду
type Command struct {
	Type  CommandType
	Key   string
	Value string // Значение нужно только для SET
}

// Parser – интерфейс парсинга строки в Command
type Parser interface {
	Parse(input string) (Command, error)
}

// parser – конкретная реализация Parser
type parser struct{}

// NewParser – конструктор для parser
func NewParser() Parser {
	return &parser{}
}

// Parse – парсит строку и возвращает структуру команды
func (p *parser) Parse(input string) (Command, error) {
	// Разбиваем входную строку по пробелам
	tokens := splitByWhitespace(input)

	if len(tokens) == 0 {
		return Command{}, errors.New("empty command")
	}

	switch strings.ToUpper(tokens[0]) {
	case "SET":
		if len(tokens) < 3 {
			return Command{}, errors.New("SET command requires 2 arguments: key and value")
		}
		return Command{
			Type:  SET,
			Key:   tokens[1],
			Value: tokens[2],
		}, nil
	case "GET":
		if len(tokens) < 2 {
			return Command{}, errors.New("GET command requires 1 argument: key")
		}
		return Command{
			Type: GET,
			Key:  tokens[1],
		}, nil
	case "DEL":
		if len(tokens) < 2 {
			return Command{}, errors.New("DEL command requires 1 argument: key")
		}
		return Command{
			Type: DEL,
			Key:  tokens[1],
		}, nil
	default:
		return Command{}, errors.New("unknown command")
	}
}

// splitByWhitespace – вспомогательная функция, которая разделяет строку по любым пробельным символам
func splitByWhitespace(input string) []string {
	fields := []string{}
	current := strings.Builder{}

	for _, r := range input {
		if unicode.IsSpace(r) {
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields
}
