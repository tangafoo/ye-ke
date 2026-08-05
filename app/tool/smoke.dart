//ignore_for_file: avoid_print
import 'package:byakugan/api/ask_client.dart';

void main() async {
  final client = AskClient(baseUrl: 'http://localhost:8080');
  await for (final e in client.ask('')) {
    print(e);
  }
}
