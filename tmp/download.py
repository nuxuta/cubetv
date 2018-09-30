import requests
import subprocess
import sys
import arrow
import os

id = sys.argv[1]

response = requests.get("https://www.cubetv.sg/studio/info?cube_id=" + id).json()

gid = response["data"]["gid"]
nick_name = response["data"]["nick_name"]
game_title = response["data"]["gameTitle"]
print(response)

# folder = game_title
# if not os.path.exists(folder):
#     os.makedirs(folder)

response = requests.get("https://www.cubetv.sg/studioApi/getStudioSrcBySid?videoType=1&https=1&sid=" + gid).json()
print(response)

if response["code"] != 1:
    print(response)
    exit()
video_src = response["data"]["video_src"]

file_name = ("%s_%s_%s_%s" % (game_title, id, nick_name, arrow.now().format('YYYYMMDD_HHmmss'))).replace(":","").replace(" ","")
executable_file = os.path.join(os.getcwd(),  file_name) + ".sh"

f = open(executable_file, "w+")
f.write("#!/bin/sh\n")
f.write("PLAYLIST=%s\n" % video_src)
f.write("OUTPUT=%s.mp4\n" % file_name)
f.write('ffmpeg -i "$PLAYLIST" -c copy -bsf:a aac_adtstoasc "$OUTPUT"\n')
f.close()

subprocess.call(['chmod', '+x', executable_file])

# log_file = open(executable_file + ".log", "wb", 0)
p = subprocess.call([executable_file])
os.remove(executable_file)
