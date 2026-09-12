import unittest

from sources.telegram.channel_role import infer_channel_role
from sources.telegram.hosting_incident import has_hosting_incident_pain, is_hosting_channel_hint
from sources.telegram.pain import message_has_tracker_pain
from sources.telegram.prefilter import should_emit_message
from sources.telegram.pwa_pain import has_pwa_pain_signal, is_pwa_channel_hint
from sources.telegram.vendor_support import vendor_support_emit_allowed


class H3H7Test(unittest.TestCase):
    def test_pwa_pain(self) -> None:
        self.assertTrue(has_pwa_pain_signal("postback from app not firing in keitaro"))
        self.assertTrue(is_pwa_channel_hint("pwa_group_support", "PWA.Group client chat"))

    def test_hosting_incident(self) -> None:
        self.assertTrue(has_hosting_incident_pain("AlexHost 502 on keitaro upstream"))
        self.assertTrue(is_hosting_channel_hint("alexhost_chat", "AlexHost support"))

    def test_vendor_support_policy(self) -> None:
        buyer = "postback failing on keitaro after redirect change"
        promo = "agency accounts for sale, warmed BMs in stock"
        self.assertTrue(vendor_support_emit_allowed(buyer, channel_role="vendor_support"))
        self.assertFalse(vendor_support_emit_allowed(promo, channel_role="vendor_support"))
        self.assertTrue(
            should_emit_message(buyer, "media_buyer_ops", channel_role="vendor_support")
        )
        self.assertFalse(
            should_emit_message(promo, "media_buyer_ops", channel_role="vendor_support")
        )

    def test_infer_channel_role(self) -> None:
        self.assertEqual(
            infer_channel_role("pwa_group_support", "PWA client chat", ""),
            "vendor_support",
        )
        self.assertEqual(infer_channel_role("affhub", "Affiliate hub", ""), "buyer_supergroup")

    def test_pwa_message_pain_path(self) -> None:
        self.assertTrue(
            message_has_tracker_pain("webview stuck, sub_id not in keitaro after PWA install")
        )


if __name__ == "__main__":
    unittest.main()
