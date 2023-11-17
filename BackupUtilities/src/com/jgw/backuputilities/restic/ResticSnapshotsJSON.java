package com.jgw.backuputilities.restic;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import com.google.gson.Gson;

public class ResticSnapshotsJSON {

	public static void main(String[] args) throws IOException {

		Path outputDir = Paths.get("e:\\delme\\restic-ouput");
		Files.createDirectories(outputDir);

		Gson gson = new Gson();

		Path snapshotsJSON = Paths.get("Z:\\Restic-Laptop\\rpi4\\snapshots.txt");

		String fileContents = Files.readString(snapshotsJSON);

		List<Map<String, Object>> list = gson.fromJson(fileContents, ArrayList.class);

		String output = "@echo off" + System.lineSeparator();

		for (Map<String, Object> m : list) {
			System.out.println("--------------");

			m.forEach((k, v) -> {
				System.out.println(k + " - " + v);
			});

			String shortID = (String) m.get("short_id");

			String time = (String) m.get("time");

			String date = time.substring(0, time.indexOf("T"));

			String dirName = date + "-" + shortID;

			Path subDir = outputDir.resolve(dirName);
			Files.createDirectories(subDir);

			String[] toExclude = new String[] {};

//			String[] toExclude = new String[] { ".git", "*.zip", "*.jar", "*.index", "websphere-developer-tools",
//					"cloud-tools", "target", "*.war", "*.webm", "*.bson", "*.trec", "*.tar.gz", "*.tgz", "*.fdt",
//					"otp-*", "*.tis", ".classCache", "*.mp4" };

			String[] toInclude = new String[] { "*.ods", "*.xls", "*.txt", "*.doc", "*.docx", "*.json", "*.bat",
					"*.odp", "*.odt" };

			output += "call Z:\\Restic-Laptop\\rpi4\\delme.bat restore --target \"" + (subDir.toString()) + "\" ";

			for (String exclude : toExclude) {
				output += "-e \"" + exclude + "\" ";
			}

			for (String include : toInclude) {
				output += "-i \"" + include + "\" ";
			}

			output += shortID + System.lineSeparator();

		}

		Files.writeString(outputDir.resolve("run.bat"), output);
	}

}
