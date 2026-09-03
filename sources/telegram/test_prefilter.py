import unittest

from sources.telegram.prefilter import (
    channel_icp_relevant,
    is_spam_message,
    should_emit_message,
)


class PrefilterTest(unittest.TestCase):
    def test_spam_rejected(self) -> None:
        self.assertTrue(is_spam_message("join our channel for vip signals"))

    def test_pain_emitted(self) -> None:
        self.assertTrue(
            should_emit_message("voluum alternative postback failing badly")
        )

    def test_channel_icp_relevant(self) -> None:
        self.assertTrue(
            channel_icp_relevant(["Igaming affiliate media buying tracker"])
        )

    def test_channel_spam_not_relevant(self) -> None:
        self.assertFalse(
            channel_icp_relevant(["VIP signal course mentorship paid tips only"])
        )

    def test_job_tutorial_noise_rejected(self) -> None:
        self.assertFalse(
            should_emit_message("we are hiring a media buyer apply now")
        )
        self.assertTrue(
            should_emit_message("hiring post but voluum postback keeps failing")
        )
        self.assertFalse(
            should_emit_message("step by step tutorial how to build funnels")
        )

    def test_soak_invite_tutorial_noise_rejected(self) -> None:
        self.assertFalse(
            should_emit_message(
                "Мануал: как работать с трекером Keitaro. Подготовили для вас мануал."
            )
        )
        self.assertFalse(
            should_emit_message(
                "Мастхев скіли в Claude для арбітражника-вайбкодера. claude keitaro."
            )
        )

    def test_programmatic_noise_rejected(self) -> None:
        self.assertFalse(
            should_emit_message(
                "Head of programmatic display buying CPM on openRTB SSP stack"
            )
        )
        self.assertTrue(
            should_emit_message(
                "openrtb postback failing on voluum migration need alternative"
            )
        )

    def test_agency_outreach_soak_rejected(self) -> None:
        self.assertFalse(
            should_emit_message(
                "Hey I help eCommerce, Brand, enterprise & affiliate businesses "
                "grow through effective Ads campaign Google, Meta, Tiktok, Snapchat "
                "and native ads etc, and setup a smart tracking + automation "
                "Cloaking GoHighLevel | RedTrack | Zapier. Open to connect."
            )
        )
        self.assertFalse(
            should_emit_message(
                "Quick question for anyone running crypto or gambling ads: "
                "What's your biggest headache right now - constant account bans "
                "or slow approvals? I help solve both by providing: Verified ad "
                "accounts ready to scale Cloaking systems built for longevity "
                "Clean white pages + converting UGC scripts Drop your main pain "
                "below or DM me privately if you want real solutions that are "
                "working for others."
            )
        )

    def test_buyer_postback_question_emitted(self) -> None:
        self.assertTrue(
            should_emit_message(
                "Hello group, Is there an option to resend a postback with an "
                "updated payout? Currently, conversions are being tracked via "
                "postback and are correctly reflected in Voluum."
            )
        )
        self.assertTrue(
            should_emit_message(
                "Hi Admin, does Voluum support for MediaGo, Outbrain, Taboola? "
                "Cause we have clients looking for dedicated support for tracking "
                "and cloaking."
            )
        )


if __name__ == "__main__":
    unittest.main()
