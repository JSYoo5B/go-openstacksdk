# Internal JMESPath engine

This is an adapted source fork of
[jmespath/go-jmespath v0.4.0](https://github.com/jmespath/go-jmespath/tree/v0.4.0).
The original copyright is James Saryerwinnie, 2015. The exact upstream
Apache-2.0 application notice is retained in `LICENSE` here. The complete
license terms are included in the repository root `LICENSE`; see
`../../THIRD_PARTY_NOTICES.md` for the distribution notices. The upstream grammar, Pratt parser,
lexer, AST and generated token names are retained with the changes below.
There is no external process or runtime dependency.

The selected Python comparison reference is
[jmespath.py 1.0.1](https://github.com/jmespath/jmespath.py/tree/1.0.1).
That tag satisfies the pinned openstacksdk dependency `jmespath>=0.9.0`;
openstacksdk does not pin an exact installed JMESPath version. It is an
independently selected reference, not an additional SDK source pin.

`Search(expression string, data any) (any, error)` accepts ordinary JSON
trees: `map[string]any`, `[]any`, `json.Number`, `bool`, `string`, and `nil`.
Callers decode JSON with `json.Decoder.UseNumber`. Struct reflection, native
integer/float values, custom mappings and cyclic Go objects are outside this
internal input representation. Backtick JSON literals also use `UseNumber`.
`Compile` and `MustCompile` retain the full upstream APIs; `MustCompile`
deliberately panics on a syntax error. Compiled literal arrays/maps are copied
on evaluation so results cannot modify a later compiled literal evaluation.
Input arrays/maps are never mutated by sorting or other functions; ordinary
field/projection results can still reference input values.

The full grammar supports fields, indices and slices, array/object/filter
projections, flattening, current-node and expression references, boolean
operators and short circuiting, comparisons, pipes, multiselect lists/hashes,
raw strings and JSON literals. The selected Python reference's deprecated
unquoted backtick strings are supported. Very large collection indices and
slice bounds/steps saturate at Go collection boundaries, preserving their
out-of-range/clamping behavior without integer overflow. Syntax and quoted
strings use Go's UTF-8/JSON representation and byte error offsets.

All 26 standard functions are retained: `abs`, `avg`, `ceil`, `contains`,
`ends_with`, `floor`, `join`, `keys`, `length`, `map`, `max`, `max_by`, `merge`,
`min`, `min_by`, `not_null`, `reverse`, `sort`, `sort_by`, `starts_with`, `sum`,
`to_array`, `to_number`, `to_string`, `type` and `values`. Function arity and
types, including every variadic argument, are checked before handlers.
By-key functions validate singleton inputs and retain child errors. Sorting
is stable and copies the source sequence. Unicode string length/reversal use
runes; they do not interpret grapheme clusters.

Comparisons, equality, numeric min/max and numeric sorting/by-key sorting
preserve exact decimal values, including integers above 2^53 and arbitrary
size exponent text. Ordering does not expand exponent-sized coefficients or
convert to `float64`. Scalar booleans and numbers are distinct. Complete
array/object equality follows the selected Python reference's ordinary
recursive equality, where nested `true == 1` and `false == 0`. Array `contains`
also uses ordinary recursive equality and never compares Go maps/slices with
the `==` operator. String ordering is supported; ordering a number and a
string raises an error. Other noncomparable type pairs yield JMESPath null.
JMESPath numerical zero remains truthy.

`abs`, `ceil` and `floor` operate on exact decimals. Integral `sum` preserves
integer precision and may retain compact scientific notation. Integer sums
that require more than **1,048,576 coefficient digits** return an explicit
arithmetic error. A compact same-exponent integer can remain exact without
expanding its exponent. Numeric ordering has no such allocation bound and
never substitutes a rounded comparison.

Nonintegral `sum`, `avg`, and floating numeric-string conversion have an
explicit finite IEEE-754 arithmetic boundary. Underflow may round to zero;
overflow and NaN/infinity results are errors. Integral numeric strings retain
exact integers, including Unicode decimal digits and Python-style numeric
separators; floating strings use finite `float64` conversion. Hexadecimal
floating strings are not accepted as Python numeric strings. JSON decimal
spelling and exact-decimal arithmetic can differ from Python's binary float
rounding or integer/float text distinction. `to_string` emits compact ASCII
JSON with exact `json.Number` text; it does not silently round large integers.

Object key traversal is deterministic **lexical order** for `keys`, `values`,
object projections and JSON serialization. This is an explicit Go/JMESPath
unordered-object boundary relative to Python dictionary insertion order.
The cloud helper's separate ordered dictionary-filter policy does not use
this engine traversal as its filter key order.

Local corrections include every swallowed interpreter/parser child error,
the lexer raw-string EOF and U+0080 boundary bugs, function argument grammar,
object projection's extra nil-prefix bug, variadic type checks, container-safe
`contains`, singleton by-key validation, and nonmutating stable sorting.
Projection and boolean short circuit behavior stays lazy: unvisited runtime
errors remain unvisited, while parser errors always occur before evaluation.
Extra slice colons, repeated bounds without a separating colon, omitted function
argument commas and trailing commas are rejected according to the standard
grammar. The Python 1.0.1 parser accepts some of those malformed expressions;
this fork's stricter syntax is an explicit correction, rather than a claim that
the Python parser rejects every same spelling.
This engine does not implement Python Resource classes, descriptors,
connection location, custom functions or Python object identity/equality.
