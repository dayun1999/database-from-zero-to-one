# go语言实现数据库,从0到1

参考来源在[这里](https://notes.eatonphil.com/database-basics.html) </br>
PS: 代码补全推荐使用`VScode的TabNine插件`

## 第一章: SELECT,INSERT,CREATE,和命令交互式REPL的实现

第一阶段,将SQL源映射到一个token列表(也就是词法解析),然后我们再调用解析函数来找到单个SQL语句(例如SELECT).这些解析函数反过来再会调用它们的辅助函数来递归地找到可被解析的块、关键字、符号(比如括号)、标识符(比如数据表的名字)和数字或者字符串字面量.</br>
然后我们会写一个in-memory的后端基于AST(抽象语法树)来实现具体操作</br>
最后我们会写一个交互式命令行来接收SQL语句并把它传给后端

## 第二章

