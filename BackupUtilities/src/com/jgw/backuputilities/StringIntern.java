package com.jgw.backuputilities;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Rather than reading the javadocs on String.intern(), I wrote this :P. Thread
 * safe.
 */
public class StringIntern {

	private final Map<String, Long> stringToLongMap = new HashMap<>();
	private final Map<Long, String> longToStringMap = new HashMap<>();

	private final AtomicLong nextId_synch = new AtomicLong(-1l);

	@SuppressWarnings("unused")
	private long totalCharacters_synch_nextId = 0l;

	private final boolean caseSensitive;

	public StringIntern(boolean caseSensitive) {
		this.caseSensitive = caseSensitive;
	}

	public Long getOrCreateId(String str) {
		Long res = getStringId(str);
		if (res != null) {
			return res;
		}

		long nextId;
		synchronized (nextId_synch) {
			nextId = nextId_synch.incrementAndGet();
			totalCharacters_synch_nextId += str.length();
		}

		if (caseSensitive) {
			stringToLongMap.put(str, nextId);
		} else {
			stringToLongMap.put(str.toLowerCase(), nextId);
		}
		longToStringMap.put(nextId, str);

		return nextId;

	}

	public Long getStringId(String str) {

		Long res;
		if (caseSensitive) {
			res = stringToLongMap.get(str);
		} else {
			res = stringToLongMap.get(str.toLowerCase());
		}

		return res;
	}

	public String getStringFromLong(long id) {
		String res = longToStringMap.get((Long) id);
		if (res == null) {
			throw new RuntimeException("ID not found: " + id);
		}
		return res;
	}

}
