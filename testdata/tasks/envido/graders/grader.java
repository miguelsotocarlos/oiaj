import java.io.*;
import java.lang.*;
import java.util.*;

public class grader {
    private static String nextLine = "";
    private static int nextIndex = 0;
    private static BufferedReader reader;
    
    private static String readToken() throws IOException {
        while (true) {
            while (nextIndex < nextLine.length() && nextLine.charAt(nextIndex) == ' ') nextIndex++;
            if (nextIndex == nextLine.length()) {
                nextLine = reader.readLine();
                nextIndex = 0;
            } else {
                break;
            }
        }
        int baseIndex = nextIndex++;
        while (nextIndex < nextLine.length() && nextLine.charAt(nextIndex) != ' ') nextIndex++;
        return nextLine.substring(baseIndex, nextIndex);
    }
    
    public static void main(String [] args) throws IOException {
        reader = new BufferedReader(new InputStreamReader(System.in));
        try (PrintWriter writer = new PrintWriter(new BufferedWriter(new OutputStreamWriter(System.out)))) {
            int numero1;
            numero1 = Integer.parseInt(readToken());
            String palo1;
            palo1 = (readToken());
            int numero2;
            numero2 = Integer.parseInt(readToken());
            String palo2;
            palo2 = (readToken());
            int numero3;
            numero3 = Integer.parseInt(readToken());
            String palo3;
            palo3 = (readToken());
            int returnedValue;
            returnedValue = envido.envido(numero1, palo1, numero2, palo2, numero3, palo3);
            writer.println(returnedValue);
        }
    }
}
