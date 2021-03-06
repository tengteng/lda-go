package lda

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// const kMaxCorpusFileLineLength = 1024 * 1024

// Document contains some unique words, each has one or more
// occurrences in this document.  Each occurrence has a topic
// assignment, where topic is an integer from 0 to K-1.
//
// wordtopics_indices maintains a map from each unique word in
// the document to a sequence of topic assignments.  For
// example:
//
// unique_words:          WORD1    WORD2  WORD3
// wordtopics_index:      |        |      |
// wordtopics:            0 3 4 0  0 3    1
//
type Document struct {
	unique_words       []string
	wordtopics_indices []int
	wordtopics         []int
	topic_histogram    Histogram
}

type Corpus []*Document

type WordIterator struct {
	doc               *Document
	unique_word_index int // Index in doc.unique_words.
	word_topic_index  int // Index in doc.wordtopics.
}

func NewWordIterator(d *Document) (iter *WordIterator, err error) {
	if d == nil {
		return nil, errors.New("NewWordIterator with a nil *Document")
	}
	if !d.IsValid() {
		return nil, errors.New(
			"NewWordIterator with an invalid Document")
	}
	iter = &WordIterator{d, 0, 0}
	return
}

func (iter WordIterator) Done() bool {
	if iter.unique_word_index > len(iter.doc.unique_words) {
		panic(fmt.Sprintf("unique_word_index = %d, len(iter.doc.unique_words) = %d",
			iter.unique_word_index, len(iter.doc.unique_words)))
	}
	return iter.unique_word_index == len(iter.doc.unique_words)
}

func (iter *WordIterator) Next() {
	if iter.Done() {
		panic("Must not call Next() when Done() is true.")
	}
	iter.word_topic_index++
	if iter.word_topic_index >= len(iter.doc.wordtopics) ||
		iter.word_topic_index >=
			iter.doc.wordtopics_indices[iter.unique_word_index+1] {
		iter.unique_word_index++
	}

}

func (iter WordIterator) Topic() int {
	if iter.Done() {
		panic("Must not call Next() when Done() is true.")
	}
	return iter.doc.wordtopics[iter.word_topic_index]
}

func (iter *WordIterator) SetTopic(new_topic int) {
	if iter.Done() {
		panic("Must not call Next() when Done() is true.")
	}
	if new_topic < 0 {
		panic("new_topic is less than 0")
	}
	if new_topic >= len(iter.doc.topic_histogram) {
		panic(fmt.Sprintf("new_topic (%d) > iter.doc.topic_histogram (%d)",
			new_topic, iter.doc.topic_histogram))
	}
	iter.doc.topic_histogram[iter.Topic()]--
	iter.doc.topic_histogram[new_topic]++
	iter.doc.wordtopics[iter.word_topic_index] = new_topic
}

func (iter WordIterator) Word() string {
	if iter.Done() {
		panic("Must not call Next() when Done() is true.")
	}
	return iter.doc.unique_words[iter.unique_word_index]
}

var SymbolsRegexp *regexp.Regexp = regexp.MustCompile(";|\\.|,|\\?|!|\"|:")

func RemoveTailingSymbols(word string) string {
	return SymbolsRegexp.ReplaceAllString(word, "")
}

// Parse a text string, words seprated by whitespaces, and create a
// Document instance.  In order to initialize topic_histogram, this
// function requires the number_of_topics.
func NewDocument(text string, num_topics int) (doc *Document, err error) {
	if num_topics <= 1 {
		return nil, errors.New("num_topics must be >= 2")
	}

	words := strings.Fields(strings.ToLower(RemoveTailingSymbols(text)))
	if len(words) <= 1 {
		return nil, errors.New("Document less than 2 words:" + text)
	}
	sort.Strings(words)

	doc = new(Document)
	doc.wordtopics = make([]int, len(words))
	doc.unique_words = make([]string, 0)
	doc.wordtopics_indices = make([]int, 0)
	doc.topic_histogram = make([]int, num_topics)
	doc.topic_histogram[0] = len(words)

	prev_word := ""
	for i := 0; i < len(words); i++ {
		if words[i] != prev_word {
			prev_word = words[i]
			doc.unique_words = append(doc.unique_words, words[i])
			doc.wordtopics_indices = append(doc.wordtopics_indices, i)
		} else {
			doc.wordtopics_indices = append(doc.wordtopics_indices,
				doc.wordtopics_indices[len(doc.wordtopics_indices)-1])
		}
	}

	if !doc.IsValid() {
		return nil, errors.New("Document is invalid")
	}
	return
}

func (d Document) IsValid() bool {
	return len(d.unique_words) >= 1 &&
		len(d.wordtopics_indices) == len(d.wordtopics) &&
		len(d.wordtopics) >= 2 &&
		len(d.topic_histogram) >= 2
}

func (d Document) Length() int {
	return len(d.wordtopics)
}

func NewCorpus() *Corpus {
	return &Corpus{}
}

func LoadCorpus(filename string, num_topics int) (corpus *Corpus, err error) {
	file, err := os.OpenFile(filename, 0, 0)
	if err != nil {
		return nil, errors.New("Cannot open file: " + filename)
	}
	defer file.Close()

	corpus = NewCorpus()
	reader := bufio.NewReader(file)
	l, is_prefix, err := reader.ReadLine()
	for err == nil {
		line := string(l)

		if is_prefix {
			return nil, errors.New("Encountered a long line:" + line)
		}

		if len(l) > 15 { // skip short lines
			doc, err := NewDocument(line, num_topics)
			if err == nil {
				*corpus = append(*corpus, doc)
			} else {
				panic("Cannot create document from: " + line + " due to " + err.Error())
			}
		}

		l, _, err = reader.ReadLine()
	}

	if err != io.EOF {
		return nil, errors.New("Error reading: " + filename + err.Error())
	}
	return corpus, nil
}
