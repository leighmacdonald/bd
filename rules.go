package main

import (
	"github.com/leighmacdonald/bd/model"
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

func (rules ruleListCollection) FindMatch(player model.PlayerState, match *MatchedPlayerList) bool {
	for _, ruleInfo := range rules {
		for _, ruleSet := range ruleInfo.Rules {
			if ruleSet.Triggers.UsernameTextMatch != nil {
				for _, nameTrigger := range ruleSet.Triggers.UsernameTextMatch.Patterns {
					switch ruleSet.Triggers.UsernameTextMatch.Mode {
					case modeEqual:
						{
							if matchStrings(nameTrigger, player.Name, ruleSet.Triggers.UsernameTextMatch.CaseSensitive) {
								return true
							}
						}
					case modeContains:
						{
							if ruleSet.Triggers.UsernameTextMatch.CaseSensitive {
								if strings.Contains(player.Name, nameTrigger) {
									return true
								}
							} else {
								if strings.Contains(strings.ToLower(player.Name), strings.ToLower(nameTrigger)) {
									return true
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}
