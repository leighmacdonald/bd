package main

import (
	"github.com/leighmacdonald/bd/model"
	"log"
	"regexp"
	"strings"
)

type ruleListCollection []TF2BDRules

func matchStrings(pattern string, subject string, caseSensitive bool) bool {
	if caseSensitive {
		return pattern == subject
	} else {
		return strings.ToLower(pattern) == strings.ToLower(subject)
	}
}

func checkText(mode textMatchMode, pattern string, value string, caseSensitive bool) bool {
	switch mode {
	case textMatchModeStartsWith:
		return strings.HasPrefix(value, pattern)
	case textMatchModeEndsWith:
		return strings.HasSuffix(value, pattern)
	case textMatchModeEqual:
		return matchStrings(pattern, value, caseSensitive)
	case textMatchModeContains:
		if caseSensitive {
			return strings.Contains(value, pattern)
		} else {
			return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
		}
	case textMatchModeWord:
		inputWords := strings.Split(value, "")
		word := value
		if !caseSensitive {
			inputWords = strings.Split(strings.ToLower(value), " ")
			word = strings.ToLower(word)
		}
		for _, iw := range inputWords {
			if iw == word {
				return true
			}
		}
		return false
	case textMatchModeRegex:
		matched, errMatch := regexp.MatchString(pattern, value)
		if errMatch != nil {
			log.Printf("Failed to run regex text match: %v", errMatch)
			return false
		}
		return matched
	}
	return false
}

func (rules ruleListCollection) FindMatch(player model.PlayerState, match *MatchedPlayerList) bool {
	for _, ruleInfo := range rules {
		for _, ruleSet := range ruleInfo.Rules {
			if ruleSet.Triggers.AvatarMatch != nil {
				// TODO
			}
			//if ruleSet.Triggers.ChatMsgTextMatch != nil {
			//	for _, chatTrigger := range ruleSet.Triggers.ChatMsgTextMatch.Patterns {
			//		if checkText(ruleSet.Triggers.ChatMsgTextMatch.Mode, chatTrigger, player) {
			//			return true
			//		}
			//	}
			//}
			if ruleSet.Triggers.UsernameTextMatch != nil {
				for _, nameTrigger := range ruleSet.Triggers.UsernameTextMatch.Patterns {
					if checkText(ruleSet.Triggers.UsernameTextMatch.Mode, nameTrigger, player.Name, ruleSet.Triggers.UsernameTextMatch.CaseSensitive) {
						return true
					}
				}
			}
		}
	}
	return false
}
