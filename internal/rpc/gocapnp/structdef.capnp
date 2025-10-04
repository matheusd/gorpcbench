
@0xd9767bf36f62edd8;

using Go = import "/go.capnp";

$Go.package("gocapnp");
$Go.import("gocapnp");

struct TreeNode {
	value @0 :Int64;
	children @1 :List(TreeNode);
}

interface API {
	nop @0 () -> ( nop :Void ) ;
	add @1 (a :Int64, b :Int64) -> ( res :Int64 );
	multTree @2 (mult :Int64, tree :TreeNode) -> (res :TreeNode);
	toHex @3 (in :Data) -> (out :Data);
}
