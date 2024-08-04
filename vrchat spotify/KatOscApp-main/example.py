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

def is_file_path_empty(file_name):
    try:
        with open(file_name, 'r') as file:
            if file.read().strip():
                return False  # File is not empty
    except FileNotFoundError:
        pass  # File does not exist, we treat this as empty
    return True

def select_file_and_save_path():
    file_path_file = 'filepath.txt'

    if is_file_path_empty(file_path_file):
        root = tk.Tk()
        root.withdraw()  # we don't want a full GUI, so keep the root window from appearing

        # Show an "Open" dialog box and return the path to the selected file
        file_path = filedialog.askopenfilename()

        if file_path:  # if a file was selected
            with open(file_path_file, 'w') as f:
                f.write(file_path)
                print(f"File path saved: {file_path}")
                return file_path
        else:
            print("No file was selected")
    else:
        print("File path already exists in filepath.txt")
        
if(is_file_path_empty()):
    pass
else:
    file_path = select_file_and_save_path()




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

        time.sleep(10)  # Wait for 10 seconds before checking again

if __name__ == "__main__":
    main()
kat.stop()
