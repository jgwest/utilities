package com.jgw.backuputilities;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class CompareLocalToRestic {

	public static void main(String[] args) throws IOException {

		// restic ls --recursive -l (snapshot id) "/(target dir)" > e:\ls-output.txt

		List<String> params = new ArrayList<>(Arrays.asList(args));
		Path resticLsPath = Paths.get(params.remove(0));

		List<String> srcDirs = Arrays.asList(args);

		for (String srcDirStr : srcDirs) {
			System.out.println();
			System.out.println("* Beginning to verify '" + srcDirStr + "'");
			verifyDirectory(Paths.get(srcDirStr), resticLsPath);
		}

	}

	private static void verifyDirectory(Path srcDir, Path resticLsPath) throws IOException {

		FileEntryMap idToEntityMapNew = new FileEntryMap();

		long total;

		System.out.println("* Starting count");
		total = Files.walk(srcDir).count();
		System.out.println("* Count complete: " + total);

		idToEntityMapNew.addRootPath(srcDir);

		// -----------------------------------------------------

		System.out.println();
		System.out.println("* Stage 1:");

		Long[] count = new Long[] { 0l };

		// Stage 1: build the idToEntityMap
		Files.walk(srcDir).forEach((Path path) -> {

			// Skip the root
			if (path.toString().equals(srcDir.toString())) {
				return;
			}

			count[0]++;
			if (count[0] % 10000 == 0) {
				System.out.println((100d * (double) count[0]) / (double) total);
			}

			idToEntityMapNew.addPath(path);

		});

		// -----------------------------------------------------

		System.out.println();
		System.out.println("* Stage 2:");

		HashSet<Long> matchesFound = new HashSet<>();

		// Stage 2: Move through the restic ls list, and remove entries that we find.
		{

			long totalResticLines = Files.lines(resticLsPath).count();
			long[] resticListCount = new long[] { 0 };

			Files.lines(resticLsPath).forEach(line -> {
				resticListCount[0]++;
				if (resticListCount[0] % 10000 == 0) {
					System.out.println((100d * (double) resticListCount[0]) / (double) totalResticLines);
				}

				if (line.startsWith("snapshot ")) {
					return;
				}

				String pathStr = line.substring(line.indexOf(" /") + 1);

				Path windowsPath = CompareResticLsListToLocal.convertPathIfNeeded(pathStr);

				FileEntry match = idToEntityMapNew.findEntity(windowsPath);
				if (match == null) {
//					System.err.println("Unable to match: " + windowsPath);
					return;
				}

				matchesFound.add(match.getId());

			});
		}

		System.out.println();
		System.out.println("* Stage 3:");
		boolean atLeastOneFailure = idToEntityMapNew.getAllFileEntriesStream().map(entity -> {
			if (entity.getParent() == null) {
				return false; // Skip root
			}

			if (!matchesFound.contains(entity.getId())) {
				System.err.println("Could not find: " + entity.reconstructPath(idToEntityMapNew));
				return true;
			}

			return false;
		}).reduce((a, b) -> a || b).get();

		if (atLeastOneFailure) {
			System.out.println();
			throw new RuntimeException("At least one could not be found.");
		}

		System.out.println("* Pass");

	}

	/**
	 * FileEntryMap maintains a memory efficient list of folders/files in the target
	 * directory.
	 */
	private static class FileEntryMap {
		/** Not thread safe */

		private final StringIntern stringKeys = new StringIntern(false);

		private final Map<Long /* id from getPathId */, List<FileEntry> /* entities with id */> idToEntityMap = new HashMap<>();

		private long nextId = 0l;

		public FileEntryMap() {
		}

		public void addRootPath(Path path) {

			addPathInternal(path, path.toString(), null, stringKeys);

		}

		public void addPath(Path path) {

			// Find parent for the current path
			FileEntry parentEntity = findEntity(path.getParent());
			if (parentEntity == null) {
				throw new RuntimeException("Could not find parent for current path: " + path);
			}

			addPathInternal(path, path.getFileName().toString(), parentEntity, stringKeys);

		}

		private void addPathInternal(Path path, String pathFilename, FileEntry parentEntity, StringIntern stringKeys) {

			FileEntry pathEntity = new FileEntry(stringKeys.getOrCreateId(pathFilename), nextId++, parentEntity);
			long pathId = getPathId(path);

			// Sanity check: reconstruct path reconstructs to path
			if (parentEntity != null && !pathEntity.reconstructPath(this).equalsIgnoreCase(path.toString())) {
				throw new RuntimeException("mismatch! " + pathEntity.reconstructPath(this) + " " + path);
			}

			// Add to entity map
			List<FileEntry> entityList = idToEntityMap.getOrDefault(pathId, new ArrayList<FileEntry>());
			entityList.add(pathEntity);
			idToEntityMap.put((Long) pathId, entityList);

		}

		public FileEntry findEntity(Path pathParam) {

			String pathParamStr = pathParam.toString();

			FileEntry res;
			long paramPathId = getPathId(pathParam);

			List<FileEntry> potentialMatches = idToEntityMap.get((Long) paramPathId);

			if (potentialMatches == null) {
//				throw new RuntimeException("Unable to find potential match for '" + pathParamStr + "'");
				return null;
			}

			List<FileEntry> matches = potentialMatches.stream().filter(potentialMatch -> {

				String potentialMatchPath = potentialMatch.reconstructPath(this);
				return potentialMatchPath.equalsIgnoreCase(pathParamStr);
			}).collect(Collectors.toList());

			if (matches.size() == 0) {
//				throw new RuntimeException("Could not find match for '" + pathParam + "'");
				return null;
			}

			if (matches.size() > 1) {
				throw new RuntimeException("Too many matches");
			}
			res = matches.get(0);

			// Sanity check: reconstructed path matches actual path
			if (res.getParent() != null && !res.reconstructPath(this).equalsIgnoreCase(pathParamStr)) {
				throw new RuntimeException("Mismatch!!!!!! " + res.reconstructPath(this) + " " + pathParamStr);
			}

			return res;
		}

		public Stream<FileEntry> getAllFileEntriesStream() {
			return idToEntityMap.values().stream().flatMap(list -> list.stream());
		}

		public StringIntern getStringKeys() {
			return stringKeys;
		}

		private static long getPathId(Path path) {
			String[] components = path.toString().split(Pattern.quote("\\"));

			long directoryId = 0;
			for (int x = 0; x < components.length; x++) {
				directoryId += components[x].toLowerCase().hashCode();
			}

			return directoryId;
		}

	}

	private static class FileEntry {

		/** Non-null except for the root */
		private final FileEntry parent;

		/** Unique monotonically increasing ID */
		private final long id;

		/** Name of the file */
		private final long name;

		public FileEntry(long name, long id, FileEntry parent) {
			this.name = name;
			this.parent = parent;
			this.id = id;
		}

		public long getId() {
			return id;
		}

		public FileEntry getParent() {
			return parent;
		}

		@SuppressWarnings("unused")
		public String reconstructIdPath() {
			if (parent == null) {
				return Long.toString(name);
			} else {
				return parent.reconstructIdPath() + "\\" + name;
			}
		}

		public String reconstructPath(FileEntryMap fileEntryMap) {
			StringIntern stringKeys = fileEntryMap.getStringKeys();

			if (parent == null) {
				return stringKeys.getStringFromLong(name);
			} else {

				String parentResult = parent.reconstructPath(fileEntryMap);
				return parentResult + "\\" + stringKeys.getStringFromLong(name);

			}
		}

	}

}
