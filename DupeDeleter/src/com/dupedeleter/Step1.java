package com.dupedeleter;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;

public class Step1 {

	public static Map<Long, List<Path>> runStep(Path stepParam) {

		Map<Long, List<Path>> fileMap = new HashMap<>();

		List<Path> queue = new ArrayList<>();

		queue.add(stepParam);

		while (queue.size() > 0) {

			Path outerPath = queue.remove(0);

			try {
				Files.list(outerPath).forEach(f -> {

					if (Files.isDirectory(f)) {
						queue.add(f);

					} else if (isIncluded(f)) {
						try {
							List<Path> paths = fileMap.computeIfAbsent(Files.size(f), (k) -> new ArrayList<>());
							paths.add(f);
						} catch (IOException e) {
							throw new RuntimeException(e);
						}
					}

				});
			} catch (Exception e) {
				System.err.println("* Skipping " + outerPath);
			}

		}

		for (Iterator<Entry<Long, List<Path>>> it = fileMap.entrySet().iterator(); it.hasNext();) {
			Entry<Long, List<Path>> e = it.next();
			if (e.getValue().size() < 2) {
				it.remove();
			}
		}

		return fileMap;

	}

	public static boolean isIncluded(Path p) {
		String lfname = p.getFileName().toString().toLowerCase();

		boolean isVideo = lfname.endsWith(".mp4") || lfname.endsWith(".avi") || lfname.endsWith(".wmv")
				|| lfname.endsWith(".mpg") || lfname.endsWith(".mpeg") || lfname.endsWith(".asf")
				|| lfname.endsWith(".mkv") || lfname.endsWith(".webm");

		boolean isImage = lfname.endsWith(".png") || lfname.endsWith(".jpg") || lfname.endsWith(".jpeg")
				|| lfname.endsWith(".gif") || lfname.endsWith(".webp");

		return isVideo || isImage;
	}

}
