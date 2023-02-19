package ui

import "fyne.io/fyne/v2/data/binding"

type boundSettings struct {
	binding.Struct
}

func (s *boundSettings) getBoundStringDefault(key string, def string) binding.String {
	value, apiKeyErr := s.GetValue(key)
	if apiKeyErr != nil {
		value = def
	}
	v := value.(string)
	return binding.BindString(&v)
}

func (s *boundSettings) getBoundBoolDefault(key string, def bool) binding.Bool {
	value, apiKeyErr := s.GetValue(key)
	if apiKeyErr != nil {
		value = def
	}
	v := value.(bool)
	return binding.BindBool(&v)
}
