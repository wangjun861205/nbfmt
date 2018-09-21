# nbfmt
## Overview
nbfmt is s simple string template format package.
## Syntax

### If statement
``` 
{{ if x == 10 }}
    hello world
{{ elseif x == 5 }}
    test
{{ else }}
    byebye
{{ endif }}
```
### For statement
``` 
{{ for i, v in someSlice }}
    the index: {{ i }}, the value: {{ v }}
{{ endfor }}
{{ for k, v in someMap }}
    the key: {{ k }}, the value: {{ k }}
{{ endfor }}
```
### Switch statement
``` 
{{ switch x }}
    {{ case "foo" }}
        test
    {{ case "bar" }}
        byebye
    {{ default }}
        hello world
{{ endswitch }}
```

## Usage
``` 
src := `{{ for i, v in l }}
            {{ if v == 1.23 }}
                the index: {{ i }}, the twice value: {{ v * 2 }}
            {{ endif }}
        {{ endfor }}`
l := []float64{1.1, 1.2, 1.23}
result, err := nbfmt.Fmt(src, map[string]interface{}{"l": l})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)
```
above code snippet will output:
```
the index: 2, the twice value: 2.46
```

Any statement can be nested in any statement, e.g
```
{{ for i, v in m }}
    {{ if v.field1.value == 0 }}
        {{ swtich v.field2.value }}
            {{ case "empty" }}
                this is empty struct
            {{ case "zero" }}
                this is zero struct
            {{ default }}
                this is unknown struct
        {{ endswitch }}
    {{ endif }}
{{ endfor }}
```

## PS:
This is only a rough edition. There may be many bugs. Don't use it in product, until the stable edition releasing. If you find some bugs, you can fix them by your self or contact me.
Forgive my poor English.I have already tried my best to write this.





