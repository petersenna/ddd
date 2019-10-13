/*
 * ddd Copyright 2019 Peter Senna Tschudin <peter.senna@gmail.com>
 *
 * SPDX-License-Identifier: GPL-2.0-only
 */

/*
 * TODO:
 * - User input
 *    - only delete a word when the player types article and word correctly
 *    - allow the player to ignore special characters like ä and ß
 * - Avoid collisions and crammed lines
 * - Help the user with repeating known words and adding new ones sporadically
 * - Add game and score rules so that
 *     - There is a score such as correct articles / 10 minutes
 *     - Number of mistakes increases difficulty level
 *     - The game ends in a few minutes
 *     - Difficulty level increases dynamically based on user skills
 *     - Socre board
 * - Colors and visual
 *    - Use terminal dimensions instead of fixed values
 *    - Add a few status lines and a place for user input
 *    - yellow words that are 60% of the way to the end
 *    - red words that are 80% of the way to the end
 *    - red background if the game is close to the end
 *    - Allow to choose between multiple dictionaries
 *    - Add some visual annimation
 * - Add music
 *    - bps increases following the speed
 */

package main

import (
	"bufio"
	"bytes"
	//"fmt"
	"github.com/ansoni/termination"
	"github.com/nsf/termbox-go"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"
)

type Worte struct {
	Article string
	Word    string
}

type dddData struct {
	wordsList    []Worte
	onScreenList map[int]WordForScreen
	mutex        *sync.Mutex
	term         *termination.Termination
	statusHeight int
}

type WordForScreen struct {
	Word    Worte
	Entity  *termination.Entity
	Hits    int
	Misses  int
	dddData *dddData
}

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

func wordMovement(t *termination.Termination, e *termination.Entity, position termination.Position) termination.Position {
	position.X += 1
	return position
}

func deathCallback(term *termination.Termination, entity *termination.Entity) {

	wordForScreen := entity.Data.(WordForScreen)

	word := wordForScreen.Word.Word
	var wordIdx int = -1

	wordForScreen.dddData.mutex.Lock()
	for key, value := range wordForScreen.dddData.onScreenList {
		if value.Word.Word == word {
			wordIdx = key
			break
		}
	}
	wordForScreen.dddData.mutex.Unlock()

	if wordIdx > 0 {
		removeFromScreen(*(wordForScreen.dddData), wordIdx, true)
	}
}

func removeFromScreen(dddData dddData, wordIdx int, noWait bool) {
	var wordForScreen WordForScreen

	dddData.mutex.Lock()
	_, ok := dddData.onScreenList[wordIdx]
	if ok {
		wordForScreen = dddData.onScreenList[wordIdx]
		delete(dddData.onScreenList, wordIdx)
	}
	dddData.mutex.Unlock()

	if !ok {
		return
	}

	article := wordForScreen.Word.Article
	word := wordForScreen.Word.Word

	wordShape := termination.Shape{
		"default": []string{
			article + " " + word,
		},
	}
	wordForScreen.Entity.Shape = wordShape
	if !noWait {
		time.Sleep(250 * time.Millisecond)
	}
	wordForScreen.Entity.Die()
}

func randomRemoveFromScreen(dddData dddData) {
	var wordIdx int
	var ok bool = false

	dddData.mutex.Lock()
	if len(dddData.onScreenList) != 0 {
		ok = true
		for key, _ := range dddData.onScreenList {
			wordIdx = key
			break
		}
	}
	dddData.mutex.Unlock()

	if !ok {
		return
	}

	removeFromScreen(dddData, wordIdx, false)
}

func randomWordToScreen(dddData dddData) {

	var wordForScreen WordForScreen
	var wordIdx int

	wordMaxIdx := len(dddData.wordsList) - 1

	// Make sure to not add same word twice on screen
	for {
		wordIdx = random(0, wordMaxIdx)

		dddData.mutex.Lock()
		_, ok := dddData.onScreenList[wordIdx]
		dddData.mutex.Unlock()

		if !ok {
			break
		}
	}

	word := dddData.wordsList[wordIdx].Word

	wordShape := termination.Shape{
		"default": []string{
			"___ " + word,
		},
	}

	wordLen := len(word) + 4 // 4 is from "___ "
	maxHeight := dddData.term.Height - dddData.statusHeight - 1
	position := termination.Position{-1 * wordLen, random(0, maxHeight), 0}

	wordForScreen.Word = dddData.wordsList[wordIdx]
	wordForScreen.dddData = &dddData
	wordForScreen.Entity = dddData.term.NewEntity(position)
	wordForScreen.Entity.Shape = wordShape
	wordForScreen.Entity.DeathOnOffScreen = true
	wordForScreen.Entity.MovementCallback = wordMovement
	wordForScreen.Entity.Data = wordForScreen
	wordForScreen.Entity.DeathCallback = deathCallback

	dddData.mutex.Lock()
	dddData.onScreenList[wordIdx] = wordForScreen
	dddData.mutex.Unlock()
}

func wordAdderLoop(dddData dddData) {
	tick := time.Tick(250 * time.Millisecond)

	for {
		select {
		case <-tick:
			randomWordToScreen(dddData)
		}
	}
}

func wordRemoverLoop(dddData dddData) {
	tick := time.Tick(250 * time.Millisecond)

	for {
		select {
		case <-tick:
			randomRemoveFromScreen(dddData)
		}
	}
}

func populateWords(inputFile string) []Worte {
	var wordsList []Worte
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineFields := strings.Split(scanner.Text(), " ")

		article := lineFields[0]
		word := lineFields[1]

		if strings.Contains(word, ",") {
			word = strings.Split(word, ",")[0]
		}
		wordsList = append(wordsList, Worte{article, word})
	}

	return wordsList
}

type Status struct {
	//Typing
	TypingSkill     int    // From one finger slow to four hands two keyboards
	TypingSkillText string // A comment/label about skill level
	Words10m        int    // How many words typed in 10 minutes

	//German
	GermanSkill     int    // From one finger slow to four hands two keyboards
	GermanSkillText string // A comment/label about skill level

	//Game
	Score    int
	TimeUsed int // Time since game started

	//Words
	Typing       string // Word being typed
	Traffic      int    // How many correct words shown
	TrafficTotal int    // Word count on the dictionary
	Missed       int    // Number of missed words

}

func statusBar(dddData dddData) {
	// Typing skill: 123 (aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa)	Words 10m: 9999
	// German skill: 123 (aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa)
	// Game score: 9999                   Time elapsed: 00s
	// [__________________________________]	 Traffic 9999/9999   Missed: 99

	var statusBuf bytes.Buffer

	position := termination.Position{0, dddData.term.Height - 4, 0}
	statusBar := dddData.term.NewEntity(position)

	templateText := "{{printf `Typing skill: %3d %-32s %14s Words 10: %4d` .TypingSkill .TypingSkillText ` ` .Words10m}}\n" +
		"{{printf `German skill: %3d %-32s %28s` .GermanSkill .GermanSkillText ` `}}\n" +
		"{{printf `Game score %-4d %45s Time elapsed: %2ds` .Score ` ` .TimeUsed}}\n" +
		"{{printf `[%32s] %7s Traffic: %-4d/%-4d %7s Missed %2d` .Typing ` ` .Traffic .TrafficTotal ` ` .Missed}}"
		/*
			templateText := "Typing skill: {{.TypingSkill}} ({{.TypingSkillText}}) Words 10m: {{.Words10m}}\n" +
				"German skill: {{.GermanSkill}} ({{.GermanSkillText}})\n" +
				"Game score: {{.Score}} Time elapsed: {{.TimeUsed}}\n" +
				"[{{.Typing}}] Traffic {{.Traffic}}/{{.TrafficTotal}} Missed: {{.Missed}}"
		*/
	status := Status{
		TypingSkill:     133,
		TypingSkillText: "Not bad",
		Words10m:        1234,
		GermanSkill:     61,
		GermanSkillText: "Very slow",
		Score:           1234,
		TimeUsed:        10,
		Typing:          "das Auto",
		Traffic:         14,
		TrafficTotal:    123,
		Missed:          5,
	}

	statusTemplate, err := template.New("status").Parse(templateText)
	if err != nil {
		panic(err)
	}

	err = statusTemplate.Execute(&statusBuf, status)
	if err != nil {
		panic(err)
	}

	statusString := statusBuf.String()

	statusShape := termination.Shape{
		"default": []string{
			statusString,
		},
	}
	statusBar.Shape = statusShape
	statusBar.DefaultColor = 'W'

}

func main() {
	var dddData dddData

	dddData.onScreenList = make(map[int]WordForScreen)
	dddData.mutex = &sync.Mutex{}
	dddData.term = termination.New()
	dddData.term.FramesPerSecond = 10
	dddData.wordsList = populateWords("A1Worteliste.txt")
	dddData.statusHeight = 4

	rand.Seed(time.Now().UnixNano())

	defer dddData.term.Close()
	go dddData.term.Animate()

	termbox.SetInputMode(termbox.InputEsc)

	go statusBar(dddData)
	go wordAdderLoop(dddData)
	time.Sleep(5 * time.Second)
	go wordRemoverLoop(dddData)

mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlC {
				break mainloop
			}
		}
	}
}
