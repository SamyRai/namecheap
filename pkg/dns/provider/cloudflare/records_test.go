package cloudflare

import (
	"testing"

	"zonekit/pkg/dnsrecord"

	"github.com/stretchr/testify/require"
)

func TestCanonicalizeTXT(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain unquoted", testSPFValue, testSPFValue},
		{"single quoted segment", `"` + testSPFValue + `"`, testSPFValue},
		{"split quoted segments", `"v=spf1 " "include:example.com ~all"`, testSPFValue},
		{"escaped quote inside segment", `"say \"hi\""`, `say "hi"`},
		{"whitespace padded", `  "hello"  `, "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, canonicalizeTXT(tc.in))
		})
	}
}

func TestRecordIdentity_TXTCanonicalCompare(t *testing.T) {
	sent := dnsrecord.Record{HostName: "@", RecordType: dnsrecord.RecordTypeTXT, Address: "v=spf1 ~all"}
	returned := dnsrecord.Record{HostName: "@", RecordType: dnsrecord.RecordTypeTXT, Address: `"v=spf1 ~all"`}

	require.Equal(t, recordIdentity(sent), recordIdentity(returned),
		"a quoted TXT value returned by the API must compare equal to the unquoted value we sent")
}

func TestDiffRecords_MetadataOnlyChangeIsUpdateNotDeleteCreate(t *testing.T) {
	existing := []dnsrecord.Record{
		{ID: testRecordID1, HostName: testHostWWW, RecordType: dnsrecord.RecordTypeA, Address: testAddrA1, TTL: 300},
	}
	desired := []dnsrecord.Record{
		// Same identity (host/type/address), only TTL changed.
		{HostName: testHostWWW, RecordType: dnsrecord.RecordTypeA, Address: testAddrA1, TTL: 600},
	}

	diff := diffRecords(existing, desired)
	require.Empty(t, diff.toCreate, "unchanged identity must not be re-created")
	require.Empty(t, diff.toDelete, "unchanged identity must not be deleted")
	require.Len(t, diff.toUpdate, 1)
	require.Equal(t, testRecordID1, diff.toUpdate[0].id)
}

func TestDiffRecords_NoOpWhenIdentical(t *testing.T) {
	existing := []dnsrecord.Record{
		{ID: testRecordID1, HostName: testHostWWW, RecordType: dnsrecord.RecordTypeA, Address: testAddrA1, TTL: 300},
	}
	desired := []dnsrecord.Record{
		{HostName: testHostWWW, RecordType: dnsrecord.RecordTypeA, Address: testAddrA1, TTL: 300},
	}

	diff := diffRecords(existing, desired)
	require.Empty(t, diff.toCreate)
	require.Empty(t, diff.toUpdate)
	require.Empty(t, diff.toDelete)
}

func TestDiffRecords_CreateAndDeleteAreIndependent(t *testing.T) {
	existing := []dnsrecord.Record{
		{ID: testRecordID1, HostName: "old", RecordType: dnsrecord.RecordTypeA, Address: testAddrA1},
		{ID: "rec-2", HostName: "keep", RecordType: dnsrecord.RecordTypeA, Address: testAddrA2},
	}
	desired := []dnsrecord.Record{
		{HostName: "keep", RecordType: dnsrecord.RecordTypeA, Address: testAddrA2},
		{HostName: "new", RecordType: dnsrecord.RecordTypeA, Address: "3.3.3.3"},
	}

	diff := diffRecords(existing, desired)
	require.Len(t, diff.toDelete, 1)
	require.Equal(t, "old", diff.toDelete[0].HostName)
	require.Len(t, diff.toCreate, 1)
	require.Equal(t, "new", diff.toCreate[0].HostName)
	require.Empty(t, diff.toUpdate)
}

func TestToWire_CommentTooLongRejected(t *testing.T) {
	r := dnsrecord.Record{
		HostName:   testHostWWW,
		RecordType: dnsrecord.RecordTypeA,
		Address:    testAddrA1,
		Metadata:   map[string]string{metaComment: string(make([]byte, maxCommentLength+1))},
	}
	_, err := toWire(r)
	require.Error(t, err)
}

func TestWireRoundTrip_SRV(t *testing.T) {
	r := dnsrecord.Record{
		HostName:   "_sip._tcp",
		RecordType: dnsrecord.RecordTypeSRV,
		Priority:   10,
		Weight:     20,
		Port:       5060,
		Target:     "sip.example.com",
	}
	w, err := toWire(r)
	require.NoError(t, err)
	require.Equal(t, "SRV", w.Type)
	require.NotEmpty(t, w.Data)

	back := fromWire(w)
	require.Equal(t, r.Priority, back.Priority)
	require.Equal(t, r.Weight, back.Weight)
	require.Equal(t, r.Port, back.Port)
	require.Equal(t, r.Target, back.Target)
}

func TestWireRoundTrip_CAA(t *testing.T) {
	r := dnsrecord.Record{
		HostName:   "@",
		RecordType: dnsrecord.RecordTypeCAA,
		Address:    "letsencrypt.org",
		Metadata:   map[string]string{metaFlags: "0", metaTag: "issue"},
	}
	w, err := toWire(r)
	require.NoError(t, err)
	require.Equal(t, "CAA", w.Type)

	back := fromWire(w)
	require.Equal(t, "letsencrypt.org", back.Address)
	require.Equal(t, "issue", back.Metadata[metaTag])
	require.Equal(t, "0", back.Metadata[metaFlags])
}
