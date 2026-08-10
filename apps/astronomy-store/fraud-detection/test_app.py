import unittest
from unittest.mock import MagicMock, patch

from app import app


class CheckOrderTest(unittest.TestCase):
    def test_returns_fraud_score_logs_it_and_enriches_active_span(self):
        span = MagicMock()

        with (
            patch("app.random.randrange", return_value=42),
            patch("app.trace.get_current_span", return_value=span),
            self.assertLogs("app", level="INFO") as logs,
        ):
            response = app.test_client().post("/fraud-detection/check-order")

        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.get_json(), {"fraud_score": 42})
        self.assertEqual(logs.records[0].msg["fraud_score"], 42)
        span.set_attribute.assert_called_once_with("astronomystore.fraud_score", 42)
