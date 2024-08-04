import katosc
import time
import tkinter as tk
from tkinter import filedialog


kat =  katosc.KatOsc()

def read_song_info(file_path):
    try:
        with open(file_path, 'r') as file:
            return file.read().strip()
    except FileNotFoundError:
        print("File not found.")
        return None
    except Exception as e:
        print(f"An error occurred: {e}")
        return None

def main():
    file_path = 'songConfig.txt'  # Update with the correct file path if needed
    last_song_info = None

    while True:
        song_info = read_song_info(file_path)
        if song_info and song_info != last_song_info:
            last_song_info = song_info
            kat.set_text(song_info)  # Update the text with the new song info
            print("Song updated:", song_info)  # Optional: for debugging

        time.sleep(0.2)  # Wait for 10 seconds before checking again

if __name__ == "__main__":
    main()
kat.stop()
