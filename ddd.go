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
 * - Polishing when the list of wordsonscreen is empty
 * - Avoid collisions and crammed lines
 * - Implement the DeathCallback to handle words that reach the end of the screen
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

type WordOnScreen struct {
	Word      Worte
	WordShape termination.Shape
	Entity    *termination.Entity
	Hits      int
	Misses    int
}

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

func wordMovement(t *termination.Termination, e *termination.Entity, position termination.Position) termination.Position {
	position.X += 1
	return position
}

func randomRemoveFromScreen(term *termination.Termination, wordsList []Worte, wordsOnScreen map[int]WordOnScreen, mutex *sync.Mutex) {

	var wordOnScreen WordOnScreen
	var wordIdx int

	mutex.Lock()
	for key, value := range wordsOnScreen {
		wordOnScreen = value
		wordIdx = key
		break
	}
	mutex.Unlock()

	article := wordOnScreen.Word.Article
	word := wordOnScreen.Word.Word

	wordShape := termination.Shape{
		"default": []string{
			article + " " + word,
		},
	}
	wordOnScreen.Entity.Shape = wordShape
	time.Sleep(300 * time.Millisecond)
	wordOnScreen.Entity.Die()

	mutex.Lock()
	delete(wordsOnScreen, wordIdx)
	mutex.Unlock()
}

func randomWordToScreen(term *termination.Termination, wordsList []Worte, wordsOnScreen map[int]WordOnScreen, mutex *sync.Mutex) {

	var wordOnScreen WordOnScreen

	wordMaxIdx := len(wordsList) - 1
	wordIdx := random(0, wordMaxIdx)

	word := wordsList[wordIdx].Word

	wordShape := termination.Shape{
		"default": []string{
			word,
		},
	}

	wordLen := len(word)
	position := termination.Position{-1 * wordLen, random(0, 25), 0}

	wordOnScreen.Word = wordsList[wordIdx]
	wordOnScreen.WordShape = wordShape
	wordOnScreen.Entity = term.NewEntity(position)
	wordOnScreen.Entity.Shape = wordShape
	wordOnScreen.Entity.DeathOnOffScreen = true
	wordOnScreen.Entity.MovementCallback = wordMovement

	mutex.Lock()
	wordsOnScreen[wordIdx] = wordOnScreen
	mutex.Unlock()
}

func wordAdderLoop(wordsList []Worte, wordsOnScreen map[int]WordOnScreen, mutex *sync.Mutex, term *termination.Termination) {
	tick := time.Tick(500 * time.Millisecond)

	for {
		select {
		case <-tick:
			randomWordToScreen(term, wordsList, wordsOnScreen, mutex)
		}
	}
}

func wordRemoverLoop(wordsList []Worte, wordsOnScreen map[int]WordOnScreen, mutex *sync.Mutex, term *termination.Termination) {
	tick := time.Tick(500 * time.Millisecond)

	for {
		select {
		case <-tick:
			randomRemoveFromScreen(term, wordsList, wordsOnScreen, mutex)
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
	var wordsList []Worte
	var wordsOnScreen = make(map[int]WordOnScreen)
	var mutex = &sync.Mutex{}

	wordsList = populateWords("A1Worteliste.txt")
	rand.Seed(time.Now().UnixNano())

	term := termination.New()
	term.FramesPerSecond = 6
	defer term.Close()
	go term.Animate()
	termbox.SetInputMode(termbox.InputEsc)

	go wordAdderLoop(wordsList, wordsOnScreen, mutex, term)
	time.Sleep(6 * time.Second)
	go wordRemoverLoop(wordsList, wordsOnScreen, mutex, term)

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
