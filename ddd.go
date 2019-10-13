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

// Typing skill: 123 (aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa)	Words 10m: 9999
// German skill: 123 (aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa)
// Game score: 9999                   Time elapsed: 00s
// [__________________________________]	 Traffic 9999/9999   Missed: 99
const statusTemplateText string = "{{printf `Typing skill: %04d %-32s %11s Words/10m: %04d` .TypingSkill .TypingSkillText ` ` .Words10m}}\n" +
	"{{printf `German skill: %04d %-32s %6s Traffic: %04d / %04d` .GermanSkill .GermanSkillText ` ` .Traffic .TrafficTotal}}\n" +
	"{{printf `  Game score: %04d %38s Time elapsed(s): %04d` .Score ` ` .TimeUsed}}\n" +
	"{{printf `[%32s] %31s Missed: %04d` .Typing ` ` .Missed}}\n"

type Worte struct {
	Article string
	Word    string
}

type dddData struct {
	wordsList    []Worte
	onScreenList map[int]*WordForScreen
	mutex        *sync.Mutex
	term         *termination.Termination
	status       Status
	statusBar    *termination.Entity
	statusHeight int
}

type WordForScreen struct {
	Word    Worte
	Entity  *termination.Entity
	Hits    int
	Misses  int
	dddData *dddData
	dead    bool
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

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

func wordMovement(term *termination.Termination, entity *termination.Entity, position termination.Position) termination.Position {
	// Let's limit to 80 chars for now
	maxLen := 80
	var isDead bool

	wordForScreen := entity.Data.(*WordForScreen)

	wordForScreen.dddData.mutex.Lock()
	isDead = wordForScreen.dead
	wordForScreen.dddData.mutex.Unlock()

	word := wordForScreen.Word.Word
	wordLen := len(word)

	if isDead && position.X >= wordLen-3 {
		return position
	}

	maxLen -= (wordLen + 4) //+4 is for the "___ "

	if position.X == maxLen {
		deathCallback(term, entity)
		return position
	}

	position.X += 1
	return position
}

func deathCallback(term *termination.Termination, entity *termination.Entity) {

	wordForScreen := entity.Data.(*WordForScreen)

	word := wordForScreen.Word.Word
	var wordIdx int = -1

	wordForScreen.dddData.mutex.Lock()
	wordForScreen.Entity.DefaultColor = 'R'
	wordForScreen.dddData.mutex.Unlock()

	wordForScreen.dddData.mutex.Lock()
	for key, value := range wordForScreen.dddData.onScreenList {
		if value.Word.Word == word {
			wordIdx = key
			break
		}
	}
	wordForScreen.dddData.mutex.Unlock()

	if wordIdx > 0 {
		go removeFromScreen(wordForScreen.dddData, wordIdx, false)
	}
}

func removeFromScreen(dddData *dddData, wordIdx int, noWait bool) {
	wordForScreen := new(WordForScreen)

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

	dddData.mutex.Lock()
	wordForScreen.Entity.Shape = wordShape
	wordForScreen.dead = true
	dddData.mutex.Unlock()

	if !noWait {
		time.Sleep(1000 * time.Millisecond)
	}
	wordForScreen.Entity.Die()
}

func randomRemoveFromScreen(dddData *dddData) {
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

	dddData.mutex.Lock()
	dddData.onScreenList[wordIdx].Entity.DefaultColor = 'G'
	dddData.mutex.Unlock()

	go removeFromScreen(dddData, wordIdx, false)
}

func randomWordToScreen(dddData *dddData) {

	wordForScreen := new(WordForScreen)
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
	wordForScreen.dddData = dddData
	wordForScreen.Entity = dddData.term.NewEntity(position)
	wordForScreen.Entity.Shape = wordShape
	wordForScreen.Entity.DeathOnOffScreen = true
	wordForScreen.Entity.MovementCallback = wordMovement
	wordForScreen.Entity.Data = wordForScreen
	wordForScreen.Entity.DeathCallback = deathCallback
	wordForScreen.Entity.DefaultColor = 'W'

	dddData.mutex.Lock()
	dddData.onScreenList[wordIdx] = wordForScreen
	dddData.mutex.Unlock()
}

func wordAdderLoop(dddData *dddData) {
	tick := time.Tick(850 * time.Millisecond)

	for {
		select {
		case <-tick:
			randomWordToScreen(dddData)
		}
	}
}

func wordRemoverLoop(dddData *dddData) {
	tick := time.Tick(850 * time.Millisecond)

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

func updateStatus(dddData *dddData) {
	statusBuf := new(bytes.Buffer)

	statusTemplate, err := template.New("status").Parse(statusTemplateText)
	if err != nil {
		panic(err)
	}

	err = statusTemplate.Execute(statusBuf, dddData.status)
	if err != nil {
		panic(err)
	}

	statusString := statusBuf.String()
	dddData.mutex.Lock()
	dddData.statusBar.Shape = termination.Shape{
		"default": []string{
			statusString,
		},
	}
	dddData.mutex.Unlock()
}

func createStatusBar(dddData *dddData) {
	dddData.statusHeight = 4

	position := termination.Position{0, dddData.term.Height - dddData.statusHeight, 0}
	dddData.statusBar = dddData.term.NewEntity(position)
	dddData.statusBar.DefaultColor = 'B'

	updateStatus(dddData)

}

func main() {
	dddData := new(dddData)

	dddData.onScreenList = make(map[int]*WordForScreen)
	dddData.mutex = &sync.Mutex{}
	dddData.term = termination.New()
	dddData.term.FramesPerSecond = 8
	dddData.wordsList = populateWords("A1Worteliste.txt")

	createStatusBar(dddData)

	rand.Seed(time.Now().UnixNano())

	defer dddData.term.Close()
	go dddData.term.Animate()

	termbox.SetInputMode(termbox.InputEsc)

	go wordAdderLoop(dddData)
	time.Sleep(5 * time.Second)
	go wordRemoverLoop(dddData)

mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlC {
				break mainloop
			} /*
				if ev.Ch != 0 {
					keyPress(dddData)
				}*/
		}
	}
}
