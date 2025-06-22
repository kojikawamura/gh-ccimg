# Malformed Markdown Test

This file contains malformed markdown to test parser resilience.

## Broken Image References

![Unclosed bracket(https://example.com/unclosed.png)
![Missing closing bracket](https://example.com/missing-bracket.png
![](https://example.com/missing-alt-closing.png
![Missing URL]()
![Missing URL and brackets]
![Only text no URL or brackets

## Broken HTML Tags

<img src="https://example.com/unclosed-tag.png" alt="Unclosed
<img src="https://example.com/missing-quotes.png alt=Missing Quotes>
<img src=https://example.com/no-quotes.png alt="No quotes on src">
<img "https://example.com/malformed-attribute.png" alt="Malformed">

## Broken Reference Links

![Reference to nowhere][nonexistent]
![Another broken ref][broken]

[broken]: this-is-not-a-url
[malformed]: https://example.com/space in url.png

## Mixed Broken and Valid

![Valid image](https://example.com/valid.png)
![Broken after valid](https://example.com/broken
<img src="https://example.com/valid-html.png" alt="Valid HTML">
<img src="https://example.com/broken-html.png" alt="Broken

## Nested Brackets and Parentheses

![Image with [brackets] in alt](https://example.com/brackets.png)
![Image with (parens) in alt](https://example.com/parens.png)
![Complex [[nested]] (brackets) and (parens)](https://example.com/complex.png)

## Empty and Whitespace

![](   )
![   ](https://example.com/empty-alt.png)
![Whitespace alt   ](   https://example.com/whitespace.png   )

## Special Characters in URLs

![Special chars](https://example.com/image with spaces.png)
![More special](https://example.com/image!@#$%^&*().png)
![Unicode](https://example.com/imagé-ñoñó.png)

## Extremely Long Content

![Very long alt text that goes on and on and on and might cause issues with parsing or memory allocation if not handled properly by the markdown parser implementation and could potentially break things if there are any buffer overflow vulnerabilities or similar issues in the underlying parsing library or custom parsing code](https://example.com/long-alt.png)

![Normal alt](https://example-with-very-long-domain-name-that-might-cause-issues-with-url-parsing-or-network-requests-if-not-handled-properly.com/very/long/path/structure/that/goes/on/and/on/and/on/and/might/cause/issues/with/path/parsing/or/url/construction/especially/if/there/are/limits/on/url/length/or/path/depth/in/the/implementation.png?very=long&query=string&with=many&parameters=that&might=cause&issues=with&url=parsing&or=network&requests=if&not=handled&properly&and=could&potentially=break&things=if&there=are&any=buffer&overflow=vulnerabilities)

## Binary and Non-text Content

This line contains some binary-like data: ����������������
And some control characters: 	
 

## Multiple Consecutive Malformed

![Bad1](bad-url-1
![Bad2](bad-url-2
![Bad3](bad-url-3
<img src="bad-html-1
<img src="bad-html-2
<img src="bad-html-3

## Valid Images After Malformed (Should Still Work)

![This should work](https://example.com/should-work.png)
<img src="https://example.com/html-should-work.jpg" alt="HTML should work">

## End

End of malformed test file.