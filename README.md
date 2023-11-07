CSS parsing by attempting to follow [the specification](https://www.w3.org/TR/css-syntax-3/).

Helped along by [Tab Atkins Jr's reference implementation](https://github.com/tabatkins/parse-css/tree/main) in JS (a co-editor of the spec), and [this CSS stylesheet parser](https://github.com/lemonrock/css/tree/master) in Rust.

Written as practice for using Go, and writing parsers. No attempt has been made to be performant!

## Usage
Currently only basic example usage is provided - the code simply reads a CSS file as input, parses the input, and serializes it back out to a file.
```sh
./go/main <input-file> # Writes output to ouput/<input-file>.css
```

## Motivation
Written primarily as more practice at using Go, practice of writing a lexer and a parser, with the handrail of a specification to follow. It was also interesting to learn more about Unicode, and the internals of CSS.

## Thoughts
### Input Representation
I started with a 'standard' Go pattern of using the `bufio` package for a `Scanner` or a `Reader`, but I found early on that it was tricky to marry the API provided by these interfaces with the needs of CSS parsing, namely the need for 3 character lookahead in the tokenization stage. Since `UnreadRune` can only be called once, I couldn't find a nice way to handle the backtracking with the buffered approach and instead changed to just reading the entire file into memory.

As I found later, the [Rust package agrees with me](https://github.com/lemonrock/css/blob/b2d6a993d26c80358c4f1b3b5f867c5012b9fb2b/src/Stylesheet.rs#L137):

>  [this package] does not use a stream of bytes as parsing CSS involves going backwards and forwards a lot... CSS parsing is somewhat evil and is not particularly efficient.

To facilitate this, I ended up writing my own Reader implementation to convert a `[]byte` to `[]rune`. I later discovered I probably could have just converted to `string`, and then range-iterated out the `rune`s - but it was interesting to learn how UTF-8 is encoded.
### Matching the spec
In terms of architecting the program, the main thing I learnt was the importance of matching the terminology in the specification to the API in the code. For example, I started off using functions like `PeekRune(0)`, `PeekRune(1)`, `ConsumeRunes(-1)`, which just got confusing to use because I had to keep mentally translating the words of the specification to the functions in my code. Once I had reworked and renamed my API to match the terminology, such as `CurrentRune()`, `NextRune()`, and `ReconsumeRune()`, the coding became very simple. I used this for the `TokenStream` side right from the start, and had very minimal difficulty just working through all the sections and implementing each algorithm.

Of couse, this leaves you with a very naive implementation that does not take advantage of any real optimizations - but I knew from the start this project was more about producing something that worked at parsing CSS, rather than being the absolute fastest / most memory efficient implementation. Once you have something that works to the spec, then you can work on optimizing.
### Thoughts on Go
The last thing to note is what I found of using Go. I have only written small self projects with Go, and overall have found it a decent language to use - it's fast to develop, and being memory managed means you don't have to think too hard about what you're doing.

However, I do often find myself annoyed and held back by the simplicity (which is part of the philosphy of Go). The lack of function overloading, or default parameters, meant I had to hack about with variadic args to implement some of the algorithms. The lack of an optional type is also annoying, but you can work round it with multiple returns. But the weakness of type system is probably the most annoying thing - the lack of ADTs, or at least a way to nicely implement unions / tagged unions, and sealed enums. These things are okay in isolation when writing small self projects, but I can see it being rather annoying when writing larger libraries / APIs / applications.