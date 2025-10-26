#include<string>
extern "C" {
  int add(int a, int b) { return a + b; }
  const char* hello(){
    return "hello from cpp";
  }
}


