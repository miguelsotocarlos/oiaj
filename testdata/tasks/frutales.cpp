#include <bits/stdc++.h>
using namespace std;

int get_sq(int x, int y, int s, vector<vector<int>> &board) {
    return
        board[x][y]
        - ((x - s >= 0) ? board[x - s][y] : 0)
        - ((y - s >= 0) ? board[x][y - s] : 0)
        + ((x - s >= 0 && y - s >= 0) ? board[x - s][y - s] : 0);
}

#define ifs cin
#define ofs cout

int main() {
    ifstream ifs("frutales.in");
    ofstream ofs("frutales.out");

    int x, y;
    cin >> x >> y;

    vector<vector<char>> board(x, vector<char>(y, '.'));

    {
        int p; int f;
        cin >> p >> f;
        for(int i = 0; i < p; ++i) {
            int xc, yc;
            cin >> xc >> yc;
            xc--; yc--;
            board[xc][yc] = 'P';
        }

        for(int i = 0; i < f; ++i) {
            int xc, yc;
            cin >> xc >> yc;
            xc--; yc--;
            board[xc][yc] = 'F';
        }
    }


    vector<vector<int>> pinos(x, vector<int>(y, 0));
    vector<vector<int>> fruta(x, vector<int>(y, 0));
    for(int xp = 0; xp < x; ++xp) for(int yp = 0; yp < y; ++yp) {
        pinos[xp][yp] = 
            (xp > 0 ? pinos[xp - 1][yp] : 0)
            + (yp > 0 ? pinos[xp][yp - 1] : 0) 
            - ((xp > 0 && yp > 0) ? pinos[xp - 1][yp - 1] : 0)
            + (board[xp][yp] == 'P' ? 1 : 0);

        fruta[xp][yp] = 
            (xp > 0 ? fruta[xp - 1][yp] : 0)
            + (yp > 0 ? fruta[xp][yp - 1] : 0) 
            - ((xp > 0 && yp > 0) ? fruta[xp - 1][yp - 1] : 0)
            + (board[xp][yp] == 'F' ? 1 : 0);
    }

    int best_x;
    int best_y;
    int best_s;
    int best_n = -1;

    for(int xp = 0; xp < x; ++xp) for(int yp = 0; yp < y; ++yp) {
        int max_w = min(xp, yp) + 1;
        int lo = 0;
        int hi = max_w + 1;
        // cerr << xp << ", " << yp << endl;
        while(hi - lo > 1) {
            int mid = (hi + lo) / 2;
            if(get_sq(xp, yp, mid, pinos) == 0) {
                lo = mid;
            } else {
                hi = mid;
            }
        }

        int num_frut = get_sq(xp, yp, lo, fruta);
        if(num_frut > best_n) {
            hi = lo;
            lo = 0;
            while(hi - lo > 1) {
                int mid = (hi + lo) / 2;
                if(get_sq(xp, yp, mid, fruta) < num_frut) {
                    lo = mid;
                } else {
                    hi = mid;
                }
            }
            best_n = num_frut;
            best_s = hi;
            best_x = xp;
            best_y = yp;
        }
    }

    cout << best_x - best_s + 1 << " " << best_y - best_s + 1 << endl;
    cout << best_s << endl;
    cout << best_n << endl;
    /*
    cerr << best_x << " " << best_y << endl;
    cerr << get_sq(best_x, best_y, best_s, fruta) << endl;
    cerr << get_sq(best_x, best_y, best_s, pinos) << endl;
    cerr << get_sq(9, 5, 4, fruta) << endl;
    */
}
