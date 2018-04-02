package transcode

import (
    "bytes"
    "fmt"
    "os"
)

const (
    TC_INGEST_UNKNOWN int = 0
    TC_INGEST_AVC     int = 1
    TC_INGEST_TS      int = 2
    TC_INGEST_CHUNK   int = 3
)

const (
    TC_TARGET_HLS      int = 0 // original Apple HLS streaming (H264, MPEG2-TS)
    TC_TARGET_DASH     int = 1 // DASH standard (not implemented yet)
    TC_TARGET_MP4      int = 3 // MP4 (H264, AAC) files for download
    TC_TARGET_OGG      int = 4 // OGG (Theora, Vorbis) files for download
    TC_TARGET_WBEM     int = 5 // WebM (VP8, Vorbis) files for download
    TC_TARGET_WEBM_HLS int = 6 // WebM (VP8, Vorbis) HLS streaming
    TC_TARGET_THUMB    int = 7 // JPG thumbnail images
    TC_TARGET_POSTER   int = 8 // JPG poster images
)

// Codec Support
//
// Make sure to compile ffmpeg with support for the following codecs if you intend to
// use them as target:
//
//   - theora: Theora video encoder for OGG
//   - vorbis: Vorbis audio encoder for OGG
//   - libogg: OGG stream format
//   - libvpx: Google's VP8 encoder
//
// OSX example
// brew install libvpx libogg libvorbis theora
// brew install ffmpeg --with-theora --with-libogg --with-libvorbis --with-libvpx

const (
    // TC_ARG_HLS      string = " -f segment -codec copy -map 0 -segment_time {{.C.Transcode.Hls.Segment_length}} -segment_format mpegts -segment_list_flags +live -segment_list_type m3u8 -individual_header_trailer 1 -segment_list index.m3u8 hls/%09d.ts"
    TC_ARG_DASH     string = " "
    TC_ARG_MP4      string = " -codec copy video.mp4 "
    TC_ARG_OGG      string = " -codec:v libtheora -b:v 1200k -codec:a vorbis -b:a 128k video.ogv "
    TC_ARG_WEBM     string = " -f webm -codec:v libvpx -quality realtime -cpu-used 0 -b:v 1200k -qmin 10 -qmax 42 -minrate 1200k -maxrate 1200k -bufsize 1500k -threads 1 -codec:a libvorbis -b:a 128k video.webm "
    TC_ARG_WEBM_HLS string = " -f webm -force_key_frames expr:gte(t,n_forced*{{.C.Transcode.Webm_hls.Segment_length}}) -codec:v libvpx -quality realtime -cpu-used 0 -b:v 1200k -qmin 10 -qmax 42 -maxrate 1200k -bufsize 1500k -lag-in-frames 0 -rc_lookahead 0 -flags +global_header -codec:a libvorbis -b:a 128k -flags +global_header -map 0 -f segment -segment_list_flags +live -segment_time {{.C.Transcode.Hls.Segment_length}} -segment_format webm -flags +global_header -segment_list webm_index.m3u8 webm/%09d.webm "
    // TC_ARG_THUMB    string = " -f image2 {{.Thumb_size}} -vsync 1 -vf fps=fps=1/{{.Thumb_rate}} thumb/%09d.jpg "
    // TC_ARG_POSTER   string = " -f image2 {{.Poster_size}} -vsync 1 -vf fps=fps=1/{{.Poster_rate}} {{.Poster_skip}} -vframes {{.Poster_corrected_count}} poster/%09d.jpg "
)

// Used Ingest options
//
// -fflags +genpts+igndts+nobuffer
// -err_detect compliant
// -avoid_negative_ts 1 [bool]
// -correct_ts_overflow 1 [bool]
// -max_delay 500000 [microsec]
// -analyzeduration 500000 [microsec]
// -f mpegts -c:0 h264
// -vsync 0
// -copyts
// -copytb 1
// -fpsprobesize 15 [frames]
// -probesize 131072 [bytes] instead of analyzeduration

// Failed options
//
// -avioflags direct                        -- broke format detection
// -f mpegtsraw -compute_pcr 0              -- created an invalid MPEGTS bitstream
// -use_wallclock_as_timestamps 1 [bool]    -- broke TS timing
// -fflags +nobuffer                        -- breaks SPS/PPS detection on some TS streams from Android ffmpeg
//
// Working with FFMpeg ingest generator, but fails with Android MPEG2/TS
// " -fflags +genpts+igndts+nobuffer -err_detect compliant -avoid_negative_ts 1 -correct_ts_overflow 1 -max_delay 500000 -analyzeduration 500000 -f mpegts -c:0 h264 -vsync 0 -copyts -copytb 1 "

// Unused Options
//
// -dts_delta_threshold <n>
// -copyinkf:0
// -fflags +discardcorrupt

const (
    TC_ARG_TSIN  string = " -fflags +genpts -err_detect compliant -avoid_negative_ts 1 -fpsprobesize 15 -probesize 131072 -max_delay 0 -f mpegts -c:0 h264 -vsync 0 -copyts -copytb 1 "
    TC_ARG_AVCIN string = " -fflags +genpts+igndts -max_delay 0 -analyzeduration 0 -f h264 -c:0 h264 -copytb 0 "
)

const (
    TC_CMD_START_PROD string = "-y -v quiet "
    TC_CMD_START_DEV  string = "-y -v debug "
    TC_CMD_INPUT      string = " -i pipe:0 "
    TC_CMD_END_PROD   string = ""
    TC_CMD_END_DEV    string = ""
)

