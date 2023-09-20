#include <string>
#include <cassert>

using namespace std;

hola

int envidoScore(int numero1, string &palo1, int numero2, string &palo2)
{
    if (palo1 == palo2)
        return 20 + numero1 + numero2;
    else
        return max(numero1, numero2);
}

int envidoValue(int x)
{
    return x >= 10 ? 0 : x;
}

int envido(int numero1, string &palo1, int numero2, string &palo2, int numero3, string &palo3) {
    int nums[3] = {numero1, numero2, numero3};
    string palo[3] = {palo1, palo2, palo3};
    // CHECKS
    assert(numero1 != numero2 || palo1 != palo2);
    assert(numero1 != numero3 || palo1 != palo3);
    assert(numero2 != numero3 || palo2 != palo3);
    for (int i = 0; i<3; i++)
    {
        assert(1 <= nums[i] && nums[i] <= 12 && nums[i] != 8 && nums[i] != 9);
        assert(palo[i] == "oros" || palo[i] == "copas" || palo[i] == "espadas" || palo[i] == "bastos");
    }
    // SOLUTION
    for (int i = 0; i<3; i++)
        nums[i] = envidoValue(nums[i]);
    
    int ret = 0;
    for (int i = 0; i<3; i++)
    for (int j = 0; j<i; j++)
        ret = max(ret, envidoScore(nums[i], palo[i], nums[j], palo[j]));
    return ret;
}


#ifndef EVAL
    #include "../graders/grader.cpp"
#endif
