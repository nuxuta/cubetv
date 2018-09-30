import requests
import subprocess
import arrow
import threading
import os
import time

count_downloading = 0


def download(cube_tv_id, user, stream_info):
    global count_downloading
    print(cube_tv_id)
    lock_file = cube_tv_id + ".downloading"

    try:
        # gid = response["data"]["gid"]
        nick_name = user["data"]["nick_name"]
        game_title = user["data"]["gameTitle"]

        video_src = stream_info["data"]["video_src"]
        file_name = ("%s-%s-%s-%s" % (nick_name, game_title, cube_tv_id, arrow.now().format('YYYYMMDD_HHmmss'))) \
            .replace(":", "").replace(" ", "")
        executable_file = os.path.join(os.getcwd(), file_name) + ".sh"
        f = open(executable_file, "w+")
        f.write("#!/bin/sh\n")
        f.write("PLAYLIST=%s\n" % video_src)
        f.write("OUTPUT=%s.mp4\n" % file_name)
        f.write('ffmpeg -i "$PLAYLIST" -c copy -bsf:a aac_adtstoasc "$1"\n')
        f.close()
        subprocess.call(['chmod', '+x', executable_file])
        time.sleep(1)
        subprocess.call([executable_file, "%s-%d.mp4" % (file_name, 1)])
        time.sleep(1)
        subprocess.call([executable_file, "%s-%d.mp4" % (file_name, 2)])
        time.sleep(1)
        subprocess.call([executable_file, "%s-%d.mp4" % (file_name, 3)])
        # os.remove(executable_file)
        os.remove(lock_file)
        count_downloading -= 1
    except Exception as e:
        print("type error: " + str(e))
        os.remove(lock_file)
        # os.remove(executable_file)
        count_downloading -= 1


while True:
    try:
        with open("follows.csv", "r") as f:
            for line in f:
                cube_tv_id = line.strip()
                if count_downloading >= 1:
                    continue

                lock_file = cube_tv_id + ".downloading"
                if os.path.isfile(lock_file):
                    print("another downloading process is running")
                    continue
                user = requests.get("https://www.cubetv.sg/studio/info?cube_id=" + cube_tv_id).json()
                gid = user["data"]["gid"]

                stream_info = requests.get(
                    "https://www.cubetv.sg/studioApi/getStudioSrcBySid?videoType=1&https=1&sid=" + gid).json()
                if stream_info["code"] != 1:
                    print(stream_info)
                    continue

                count_downloading += 1
                lf = open(lock_file, "w+")
                lf.close()

                t = threading.Thread(target=download, args=(cube_tv_id, user, stream_info))
                t.start()
            f.close()
        time.sleep(20)
    except IOError as e:
        err_log = open("error.log", "w+")
        err_log.write("I/O error({0}): {1}".format(e.errno, e.strerror))
        err_log.close()
    except:  # handle other exceptions such as attribute errors
        err_log = open("error.log", "w+")
        err_log.write("System error: {0}".format(e.errno, e.strerror))
        err_log.close()



