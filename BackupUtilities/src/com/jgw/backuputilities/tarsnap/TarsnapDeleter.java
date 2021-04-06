package com.jgw.backuputilities.tarsnap;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileReader;
import java.io.FileWriter;
import java.io.IOException;
import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.Collections;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.concurrent.TimeUnit;

public class TarsnapDeleter {

	public static void main(String[] args) throws IOException {

		if (args.length != 3) {
			System.out.println("arg 1: tarsnap file");
			System.out.println("arg 2: output script file");
			System.out.println("arg 3: delete line");

			return;
		}

		File outputFile = new File(args[1]);

		File f = new File(args[0]);

		HashMap<Long, List<Entry>> monthEntries = new HashMap<>();

		{
			List<Entry> result = readEntries(f);

			long currTime = System.currentTimeMillis();

			for (Entry e : result) {

				long timeDiff = currTime - e.time;

				long daysDiff = TimeUnit.DAYS.convert(timeDiff, TimeUnit.MILLISECONDS);

				if (daysDiff <= 31) {
					System.out.println("[" + daysDiff + "] daily: " + e);
				} else if (daysDiff <= 7 * 12) {
					long weekNum = daysDiff / 7;
					System.out.println("[" + weekNum + "] weekly: " + e);

				} else {
					Calendar c = Calendar.getInstance();
					c.setTime(new Date(e.time));

					long monthNum = 100 * c.get(Calendar.YEAR) + (c.get(Calendar.MONTH) + 1);

					List<Entry> monthEntry = monthEntries.computeIfAbsent(monthNum, a -> (new ArrayList<Entry>()));
					monthEntry.add(e);

					System.out.println("[" + monthNum + "] monthly: " + e + "");
				}

			}
		}

		System.out.println("----------------------");

		List<Entry> entriesToDelete = new ArrayList<>();

		monthEntries.forEach((k, v) -> {

			if (v.size() > 1) {
				System.out.println();
				for (int x = 0; x < v.size() - 1; x++) {
					Entry curr = v.get(x);
					System.out.println("delete: " + curr);
					entriesToDelete.add(curr);
				}
				System.out.println("keep: " + v.get(v.size() - 1));
			}

		});

		System.out.println("----------------------");

		System.out.println("To delete:");
		Collections.sort(entriesToDelete);

		entriesToDelete.forEach(e -> {
			System.out.println(e);
		});

		writeScript(outputFile, entriesToDelete, args[2]);

	}

	private static final void writeScript(File outputFile, List<Entry> result, String deleteLine) throws IOException {
		FileWriter fw = new FileWriter(outputFile);
		fw.write("#!/bin/bash\n");

		for (Entry e : result) {
			fw.write(deleteLine + " " + e.name + "\n");
		}

		fw.close();

	}

	@SuppressWarnings("resource")
	private static final List<Entry> readEntries(File f) throws IOException {
		BufferedReader br = new BufferedReader(new FileReader(f));

		String str;

		List<Entry> result = new ArrayList<Entry>();

		while (null != (str = br.readLine())) {

			Entry e = parseEntry(str);
			if (e == null) {
				throw new RuntimeException("Invalid line: " + str);
			}

			result.add(e);
		}

		br.close();

		Collections.sort(result);

		return result;
	}

	private static final String GENERAL_BACKUP = "general-backup-";

	private static Entry parseEntry(String line) {
		if (!line.startsWith(GENERAL_BACKUP)) {
			return null;
		}

		Calendar cal = Calendar.getInstance();
		cal.clear();

		int c = GENERAL_BACKUP.length();

		int year = Integer.parseInt(line.substring(c, c + 4));
		c += 4;
		c++;
		cal.set(Calendar.YEAR, year);

		int month = Integer.parseInt(line.substring(c, c + 2));
		c += 2;
		c++;
		cal.set(Calendar.MONTH, month - 1);

		int day = Integer.parseInt(line.substring(c, c + 2));
		c += 2;
		// there is no c++ here; this is intentional.
		cal.set(Calendar.DAY_OF_MONTH, day);

		if (line.contains("_") && line.charAt(c) == '_') {
			c++; // increment past the _

			int hour = Integer.parseInt(line.substring(c, c + 2));
			c += 2;
			c++;
			cal.set(Calendar.HOUR_OF_DAY, hour);

			int minute = Integer.parseInt(line.substring(c, c + 2));
			c += 2;
			c++;
			cal.set(Calendar.MINUTE, minute);

			int second = Integer.parseInt(line.substring(c, c + 2));
			c += 2;
			c++;
			cal.set(Calendar.SECOND, second);

		}

		Entry e = new Entry();
		e.time = cal.getTimeInMillis();
		e.name = line;

		return e;
	}
}

class Entry implements Comparable<Entry> {
	long time;
	String name;

	@Override
	public int compareTo(Entry o) {

		return new Date(time).compareTo(new Date(o.time));

	}

	private static final DateFormat df = SimpleDateFormat.getInstance();

	@Override
	public String toString() {
		return name + " -> " + df.format(time) + " (" + time + ")";
	}

}