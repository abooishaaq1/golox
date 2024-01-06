# golox

## changes

- comments start with `#`
- `0` is falsey
- strings can be concatenated to boolean and number values
- first part of for can only have an initializer
- implements lists
- doesn't support classes

## todo

- a better way of encoding line info in chunk.go
- compiled files
- more than 256 local varibles
- lessen similar shortcomings of the compiler

## jit

[asm.go](asm/asm.go) is a simple assembler and the plan is to use it to generate machine code for the vm. The vm will be modified to support a jit mode. The jit will be a simple trace compiler that will compile a trace of the vm's execution. The trace will be compiled to machine code and then executed. The trace will be compiled to m

Resources:

- [tracemonkey](https://web.stanford.edu/class/cs343/resources/tracemonkey.pdf)
- [hotpathVM](https://www.usenix.org/legacy/events/vee06/full_papers/p144-gal.pdf)
- [intel manuals](https://www.intel.com/content/www/us/en/developer/articles/technical/intel-sdm.html)
