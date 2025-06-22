# Complex Markdown Test

This file tests various image reference formats and edge cases.

## Inline Image References

![Alt text](https://example.com/image1.png)
![Another image with special chars!@#$%](https://example.com/image2.jpg)
![](https://example.com/no-alt-text.gif)

## HTML Image Tags

<img src="https://example.com/html-image.png" alt="HTML Image">
<img src="https://example.com/html-image2.jpg" alt="HTML Image 2" width="500" />
<img src="https://example.com/html-image3.webp" />

## Reference Style Images

![Reference Image][ref1]
![Another Reference][ref2]

[ref1]: https://example.com/ref-image1.png "Reference Image 1"
[ref2]: https://example.com/ref-image2.svg "Reference Image 2"

## GitHub-Specific URLs

![GitHub User Content](https://user-images.githubusercontent.com/12345/67890/image.png)
![GitHub Assets](https://github.com/user/repo/assets/123456/screenshot.png)

## Different Image Extensions

![PNG](https://example.com/test.png)
![JPG](https://example.com/test.jpg)
![JPEG](https://example.com/test.jpeg)
![GIF](https://example.com/test.gif)
![WebP](https://example.com/test.webp)
![SVG](https://example.com/test.svg)
![BMP](https://example.com/test.bmp)
![TIFF](https://example.com/test.tiff)
![ICO](https://example.com/test.ico)

## Images in Lists

- ![List Image 1](https://example.com/list1.png)
- ![List Image 2](https://example.com/list2.jpg)

1. ![Numbered List 1](https://example.com/num1.png)
2. ![Numbered List 2](https://example.com/num2.jpg)

## Images in Tables

| Description | Image |
|-------------|-------|
| Table Image 1 | ![Table 1](https://example.com/table1.png) |
| Table Image 2 | ![Table 2](https://example.com/table2.jpg) |

## Images in Code Blocks (Should NOT be extracted)

```markdown
![Code Block Image](https://example.com/code-block.png)
```

`![Inline Code Image](https://example.com/inline-code.png)`

## Images in Blockquotes

> ![Blockquote Image](https://example.com/blockquote.png)
> 
> This is a quoted image.

## Nested Images in Details

<details>
<summary>Click to see images</summary>

![Details Image 1](https://example.com/details1.png)
![Details Image 2](https://example.com/details2.jpg)

</details>

## Duplicate Images (Should be deduplicated)

![Duplicate 1](https://example.com/duplicate.png)
![Duplicate 2](https://example.com/duplicate.png)
<img src="https://example.com/duplicate.png" alt="Duplicate 3">

## Edge Cases

![Image with query params](https://example.com/image.png?v=123&size=large)
![Image with fragment](https://example.com/image.jpg#section)
![Image with both](https://example.com/image.gif?v=1#top)

## Malformed/Invalid (Should be handled gracefully)

![Malformed reference without link][missing]
![](not-a-url)
<img src="" alt="Empty src">
<img alt="No src">

## Protocol Variations

![HTTP](http://example.com/http.png)
![HTTPS](https://example.com/https.png)
![Protocol Relative](//example.com/protocol-relative.png)

## Long URLs

![Very Long URL](https://very-long-domain-name-that-might-cause-issues.example.com/very/long/path/to/an/image/file/with/many/segments/and/a/very/long/filename-that-might-test-url-parsing-limits.png?parameter1=value1&parameter2=value2&parameter3=value3&parameter4=value4&parameter5=value5#very-long-fragment-identifier)

## Non-Latin Characters

![Unicode Image](https://example.com/ünïcødé-image.png)
![Emoji 🖼️](https://example.com/emoji-image.jpg)

## End of Test

This concludes the complex markdown test file.