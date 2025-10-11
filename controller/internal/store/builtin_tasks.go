package store

import (
	"log"
)

// InitBuiltinTaskDefs initializes predefined builtin task definitions
func InitBuiltinTaskDefs(s Store) error {
	builtinDefs := []TaskDefinition{
		{
			Name:               "builtin.echo",
			Executor:           "embedded",
			TargetKind:         "",
			TargetRef:          "",
			DefaultPayloadJSON: `{"message": "hello"}`,
			Labels: map[string]string{
				"builtin":     "true",
				"description": "回显输入内容",
			},
		},
		{
			Name:               "builtin.delay",
			Executor:           "embedded",
			TargetKind:         "",
			TargetRef:          "",
			DefaultPayloadJSON: `{"seconds": 3}`,
			Labels: map[string]string{
				"builtin":     "true",
				"description": "延迟指定秒数（默认3秒）",
			},
		},
		{
			Name:               "builtin.fail",
			Executor:           "embedded",
			TargetKind:         "",
			TargetRef:          "",
			DefaultPayloadJSON: `{}`,
			Labels: map[string]string{
				"builtin":     "true",
				"description": "故意失败（用于测试）",
			},
		},
	}

	for _, def := range builtinDefs {
		// Check if already exists
		if _, exists, _ := s.GetTaskDefByName(def.Name); !exists {
			_, err := s.CreateTaskDef(def)
			if err != nil {
				log.Printf("Warning: failed to create builtin task %s: %v", def.Name, err)
				// Continue to next builtin even if one fails
				continue
			}
			log.Printf("Initialized builtin task: %s", def.Name)
		}
	}
	return nil
}
