ddd - ddd German article trainer

# Screenshot
This is a development version that does not allow you to play yet, but shows
the main idea of the player typing the correct article for a word before the
word reaches the end of the screen.

![ddd screenshot](/docs/ddd_screenshot.gif?raw=true "ddd screenshot")

# A1Worteliste.txt
The file A1Worteliste.txt was generated with:

    $ pdf2ps A1_SD1_Wortliste_02.pdf # Pages 9-27
    $ ps2ascii A1_SD1_Wortliste_02.ps > A1_SD1_Wortliste_02.txt
    $ libreoffice A1_SD1_Wortliste_02.txt # Save as text A1_SD1_Wortliste_02_lo.txt
    
    $ cat A1_SD1_Wortliste_02_lo.txt |tr -s " "|cut -d " " -f 2-|grep "^der " > A1Worteliste.txt
    $ cat A1_SD1_Wortliste_02_lo.txt |tr -s " "|cut -d " " -f 2-|grep "^die " >> A1Worteliste.txt
    $ cat A1_SD1_Wortliste_02_lo.txt |tr -s " "|cut -d " " -f 2-|grep "^das " >> A1Worteliste.txt
    
    $ dos2unix A1Worteliste.txt

    Manually remove lines ending with -

The line formats are:

    - Article Word Example
    - Article (pl.) Word Example
    - Article Word, Plural-modifier Example
    - Article Word, Plural-modifier, Plural-modifier(lower-case), Example
