#include <iostream>
#include <string>
#include <vector>

using namespace std;

int envido(int numero1, string &palo1, int numero2, string &palo2, int numero3, string &palo3);

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);
    int numero1;
    cin >> numero1;
    string palo1;
    cin >> palo1;
    int numero2;
    cin >> numero2;
    string palo2;
    cin >> palo2;
    int numero3;
    cin >> numero3;
    string palo3;
    cin >> palo3;
    int returnedValue;
    returnedValue = envido(numero1, palo1, numero2, palo2, numero3, palo3);
    cout << returnedValue << "\n";
    return 0;
}
