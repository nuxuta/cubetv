import os.path
import os
import requests
import subprocess
import arrow


def download(id):
    lock_file = id + ".downloading"
    if os.path.isfile(lock_file):
        print("There is another downloading process is running")
        return
    # f = open(lock_file, "w+")
    # f.write("Downloading")

    response = requests.get("https://www.cubetv.sg/studio/info?cube_id=" + id).json()
    gid = response["data"]["gid"]

    response = requests.get("https://www.cubetv.sg/studioApi/getStudioSrcBySid?videoType=1&https=1&sid=" + gid).json()
    if response["code"] != 1:
        print(response)
        return 0

    video_src = response["data"]["video_src"]

    file_name = '%s-%s' % (id, arrow.now().format('YYYY-MM-DD_HHmmss'))
    f = open(file_name + '.sh', "w+")
    f.write("PLAYLIST=%s\n" % video_src)
    f.write("OUTPUT=%s.mp4\n" % file_name)
    f.write("touch %s\n" % lock_file)
    f.write('ffmpeg -i "$PLAYLIST" -c copy -bsf:a aac_adtstoasc "$OUTPUT"\n')
    f.write('rm %s\n' % lock_file)
    f.close()

    executable_file = './' + file_name + '.sh'
    subprocess.call(['chmod', '+x', executable_file])
    # p = subprocess.Popen(['bash', executable_file], subprocess.STDOUT)
    # p.wait()
    return 1


def check():
    file = open("follows.csv", "r")
    for line in file:
        uid = line.strip()
        download(uid)
    file.close()
    return


check()
