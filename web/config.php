<?php
// Aturan upload web ini. Ubah sesukamu, nggak ngaruh ke kode utama.
return [
    // Jenis file yang boleh masuk (extension familiar).
    'allow_ext' => ['jpg', 'jpeg', 'png', 'webp', 'gif', 'mp4', 'mp3'],
    // Batas ukuran per file (MB).
    'max_mb' => 50,
    // Batas CPU per file (biar 1 upload nggak ngabisin core).
    'ffmpeg_threads' => 2,
    // Batas kerja bareng (isi 0 = tanpa batas, khusus server gede).
    // Default ikut jumlah CPU mesin.
    'max_slots' => null,
    // Folder hasil bersih (relatif dari folder web/).
    'upload_dir' => __DIR__ . '/uploads',
];
