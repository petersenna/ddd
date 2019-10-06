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
	//"fmt"
	"github.com/ansoni/termination"
	"github.com/nsf/termbox-go"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
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
	position := termination.Position{-1 * wordLen, random(0, dddData.term.Height), 0}

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

func main() {
	var dddData dddData

	dddData.onScreenList = make(map[int]WordForScreen)
	dddData.mutex = &sync.Mutex{}
	dddData.term = termination.New()
	dddData.term.FramesPerSecond = 10
	dddData.wordsList = populateWords("A1Worteliste.txt")

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
			}
		}
	}
}
