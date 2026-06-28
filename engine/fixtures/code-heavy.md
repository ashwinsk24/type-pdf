# Code Examples

Here is a short inline `code` example.

```
console.log("Hello World")
```

Below is a longer code block with multiple lines:

```javascript
function fibonacci(n) {
    if (n <= 1) return n;
    let a = 0, b = 1;
    for (let i = 2; i <= n; i++) {
        let temp = a + b;
        a = b;
        b = temp;
    }
    return b;
}

for (let i = 0; i < 10; i++) {
    console.log(fibonacci(i));
}
```
